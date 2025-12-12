// Source: https://github.com/distribution/distribution

// Copyright 2014 https://github.com/distribution/distribution Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//    http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

// Package s3 provides a storagedriver.StorageDriver implementation to
// store blobs in Amazon S3 cloud storage.
//
// This package leverages the official aws client library for interfacing with
// S3.
//
// Because S3 is a key, value store the Stat call does not support last modification
// time for directories (directories are an abstraction for key, value stores)
//
// Keep in mind that S3 guarantees only read-after-write consistency for new
// objects, but no read-after-update or list-after-write consistency.
package s3

import (
	"bytes"
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"io"
	"math"
	"net/http"
	"path/filepath"
	"reflect"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/harness/gitness/registry/app/dist_temp/dcontext"
	storagedriver "github.com/harness/gitness/registry/app/driver"
	"github.com/harness/gitness/registry/app/driver/base"
	"github.com/harness/gitness/registry/app/driver/factory"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/endpoints"
	"github.com/aws/aws-sdk-go/aws/request"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/rs/zerolog/log"
)

const driverName = "s3aws"

// minChunkSize defines the minimum multipart upload chunk size
// S3 API requires multipart upload chunks to be at least 5MB.
const minChunkSize = 5 * 1024 * 1024

// maxChunkSize defines the maximum multipart upload chunk size allowed by S3.
// S3 API requires max upload chunk to be 5GB.
const maxChunkSize = 5 * 1024 * 1024 * 1024

const defaultChunkSize = 2 * minChunkSize

const (
	// defaultMultipartCopyChunkSize defines the default chunk size for all
	// but the last Upload Part - Copy operation of a multipart copy.
	// Empirically, 32 MB is optimal.
	defaultMultipartCopyChunkSize = 32 * 1024 * 1024

	// defaultMultipartCopyMaxConcurrency defines the default maximum number
	// of concurrent Upload Part - Copy operations for a multipart copy.
	defaultMultipartCopyMaxConcurrency = 100

	// defaultMultipartCopyThresholdSize defines the default object size
	// above which multipart copy will be used. (PUT Object - Copy is used
	// for objects at or below this size.)  Empirically, 32 MB is optimal.
	defaultMultipartCopyThresholdSize = 32 * 1024 * 1024
)

// listMax is the largest amount of objects you can request from S3 in a list call.
const listMax = 1000

// noStorageClass defines the value to be used if storage class is not supported by the S3 endpoint.
const noStorageClass = "NONE"

const r2Regions = "wnam, enam, weur, eeur, apac, oc"

// s3StorageClasses lists all compatible (instant retrieval) S3 storage classes.
var s3StorageClasses = []string{
	noStorageClass,
	s3.StorageClassStandard,
	s3.StorageClassReducedRedundancy,
	s3.StorageClassStandardIa,
	s3.StorageClassOnezoneIa,
	s3.StorageClassIntelligentTiering,
	s3.StorageClassOutposts,
	s3.StorageClassGlacierIr,
}

// validRegions maps known s3 region identifiers to region descriptors.
var validRegions = map[string]struct{}{}

// validObjectACLs contains known s3 object Acls.
var validObjectACLs = map[string]struct{}{}

// DriverParameters A struct that encapsulates all of the driver parameters after all values have been set.
type DriverParameters struct {
	AccessKey                   string
	SecretKey                   string
	Bucket                      string
	Region                      string
	RegionEndpoint              string
	ForcePathStyle              bool
	Encrypt                     bool
	KeyID                       string
	Secure                      bool
	SkipVerify                  bool
	V4Auth                      bool
	ChunkSize                   int64
	MultipartCopyChunkSize      int64
	MultipartCopyMaxConcurrency int64
	MultipartCopyThresholdSize  int64
	RootDirectory               string
	StorageClass                string
	UserAgent                   string
	ObjectACL                   string
	SessionToken                string
	UseDualStack                bool
	Accelerate                  bool
	LogLevel                    aws.LogLevelType
}

func GetDriverName() string {
	return driverName
}

func init() {
	partitions := endpoints.DefaultPartitions()
	for _, p := range partitions {
		for region := range p.Regions() {
			validRegions[region] = struct{}{}
		}
	}

	// Add the default Cloudflare R2 regions
	for region := range strings.SplitSeq(r2Regions, ",") {
		validRegions[strings.TrimSpace(region)] = struct{}{}
	}
	for _, objectACL := range []string{
		s3.ObjectCannedACLPrivate,
		s3.ObjectCannedACLPublicRead,
		s3.ObjectCannedACLPublicReadWrite,
		s3.ObjectCannedACLAuthenticatedRead,
		s3.ObjectCannedACLAwsExecRead,
		s3.ObjectCannedACLBucketOwnerRead,
		s3.ObjectCannedACLBucketOwnerFullControl,
	} {
		validObjectACLs[objectACL] = struct{}{}
	}

	// Register this as the default s3 driver in addition to s3aws
	factory.Register(driverName, &s3DriverFactory{})
}

// TODO: figure-out why init is not called automatically
func Register(ctx context.Context) {
	log.Ctx(ctx).Info().Msgf("registering s3 driver")
}

// s3DriverFactory implements the factory.StorageDriverFactory interface.
type s3DriverFactory struct{}

func (factory *s3DriverFactory) Create(ctx context.Context, parameters map[string]any) (
	storagedriver.StorageDriver,
	error,
) {
	return FromParameters(ctx, parameters)
}

var _ storagedriver.StorageDriver = &driver{}

type driver struct {
	S3                          *s3.S3
	Bucket                      string
	ChunkSize                   int64
	Encrypt                     bool
	KeyID                       string
	MultipartCopyChunkSize      int64
	MultipartCopyMaxConcurrency int64
	MultipartCopyThresholdSize  int64
	RootDirectory               string
	StorageClass                string
	ObjectACL                   string
	pool                        *sync.Pool
}

func (d *driver) CopyObject(ctx context.Context, srcKey, destBucket, destKey string) error {
	// Get source object info to determine size
	srcPath := strings.TrimPrefix(srcKey, "/")
	fileInfo, err := d.Stat(ctx, srcPath)
	if err != nil {
		return fmt.Errorf("failed to get source object info: %w", err)
	}

	// For objects <= threshold size, use simple copy
	if fileInfo.Size() <= d.MultipartCopyThresholdSize {
		copySource := fmt.Sprintf("/%s%s", d.Bucket, srcKey)
		_, err := d.S3.CopyObjectWithContext(ctx, &s3.CopyObjectInput{
			Bucket:     aws.String(destBucket),
			CopySource: aws.String(copySource),
			Key:        aws.String(destKey),
		})
		if err != nil {
			return err
		}
	} else {
		// For large objects, use multipart copy
		err = d.performMultipartCopy(ctx, d.Bucket, srcKey, destBucket, destKey, fileInfo.Size())
		if err != nil {
			return err
		}
	}

	// Verify the destination object exists
	_, err = d.S3.HeadObjectWithContext(ctx, &s3.HeadObjectInput{
		Bucket: aws.String(destBucket),
		Key:    aws.String(destKey),
	})
	return err
}

// performMultipartCopy performs multipart copy for large objects across buckets.
func (d *driver) performMultipartCopy(
	ctx context.Context, srcBucket, srcKey, destBucket, destKey string, objectSize int64,
) error {
	log.Ctx(ctx).Trace().Msgf("[AWS] CreateMultipartUpload: %s/%s -> %s/%s", srcBucket, srcKey, destBucket, destKey)

	// Create multipart upload
	createResp, err := d.S3.CreateMultipartUploadWithContext(
		ctx, &s3.CreateMultipartUploadInput{
			Bucket:               aws.String(destBucket),
			Key:                  aws.String(destKey),
			ContentType:          d.getContentType(),
			ACL:                  d.getACL(),
			SSEKMSKeyId:          d.getSSEKMSKeyID(),
			ServerSideEncryption: d.getEncryptionMode(),
			StorageClass:         d.getStorageClass(),
		},
	)
	if err != nil {
		return err
	}

	numParts := (objectSize + d.MultipartCopyChunkSize - 1) / d.MultipartCopyChunkSize
	completedParts := make([]*s3.CompletedPart, numParts)
	errChan := make(chan error, numParts)
	limiter := make(chan struct{}, d.MultipartCopyMaxConcurrency)

	for i := range completedParts {
		i := int64(i)
		go func() {
			limiter <- struct{}{}
			firstByte := i * d.MultipartCopyChunkSize
			lastByte := firstByte + d.MultipartCopyChunkSize - 1
			if lastByte >= objectSize {
				lastByte = objectSize - 1
			}
			log.Ctx(ctx).Trace().Msgf("[AWS] [%d] UploadPartCopy: %s/%s -> %s/%s", i, srcBucket, srcKey, destBucket,
				destKey)
			uploadResp, err := d.S3.UploadPartCopyWithContext(
				ctx, &s3.UploadPartCopyInput{
					Bucket:          aws.String(destBucket),
					CopySource:      aws.String(srcBucket + "/" + strings.TrimPrefix(srcKey, "/")),
					Key:             aws.String(destKey),
					PartNumber:      aws.Int64(i + 1),
					UploadId:        createResp.UploadId,
					CopySourceRange: aws.String(fmt.Sprintf("bytes=%d-%d", firstByte, lastByte)),
				},
			)
			if err == nil {
				completedParts[i] = &s3.CompletedPart{
					ETag:       uploadResp.CopyPartResult.ETag,
					PartNumber: aws.Int64(i + 1),
				}
			}
			errChan <- err
			<-limiter
		}()
	}

	for range completedParts {
		err := <-errChan
		if err != nil {
			// Abort the multipart upload on error
			_, abortErr := d.S3.AbortMultipartUploadWithContext(ctx, &s3.AbortMultipartUploadInput{
				Bucket:   aws.String(destBucket),
				Key:      aws.String(destKey),
				UploadId: createResp.UploadId,
			})
			if abortErr != nil {
				log.Ctx(ctx).Error().Err(abortErr).Msg("Failed to abort multipart upload")
			}
			return err
		}
	}

	log.Ctx(ctx).Trace().Msgf("[AWS] CompleteMultipartUpload: %s/%s", destBucket, destKey)
	_, err = d.S3.CompleteMultipartUploadWithContext(
		ctx, &s3.CompleteMultipartUploadInput{
			Bucket:          aws.String(destBucket),
			Key:             aws.String(destKey),
			UploadId:        createResp.UploadId,
			MultipartUpload: &s3.CompletedMultipartUpload{Parts: completedParts},
		},
	)
	return err
}

func (d *driver) BatchCopyObjects(ctx context.Context, destBucket string, keys []string, concurrency int) error {
	total := len(keys)
	sem := make(chan struct{}, concurrency)
	errCh := make(chan error, total)
	var wg sync.WaitGroup

	var mu sync.Mutex
	completed := 0

	for _, key := range keys {
		wg.Add(1)
		sem <- struct{}{}

		go func(key string) {
			defer wg.Done()
			defer func() { <-sem }()

			var err error
			for attempt := 1; attempt <= 3; attempt++ {
				err = d.CopyObject(ctx, key, destBucket, key)
				if err == nil {
					break
				}
				time.Sleep(time.Duration(100*attempt) * time.Millisecond) // basic exponential backoff
			}
			if err != nil {
				errCh <- fmt.Errorf("failed to copy key %s after %d retries: %w", key, 3, err)
				return
			}

			// Update progress
			mu.Lock()
			completed++
			log.Ctx(ctx).Info().Msgf("Progress: %d/%d copied", completed, total)
			mu.Unlock()
		}(key)
	}

	wg.Wait()
	close(errCh)

	if len(errCh) > 0 {
		return <-errCh
	}
	return nil
}

type baseEmbed struct {
	base.Base
}

// Driver is a storagedriver.StorageDriver implementation backed by Amazon S3
// Objects are stored at absolute keys in the provided bucket.
type Driver struct {
	baseEmbed
}

// FromParameters constructs a new Driver with a given parameters map
// Required parameters:
// - accesskey
// - secretkey
// - region
// - bucket
// - encrypt.
//
//nolint:gocognit
func FromParameters(ctx context.Context, parameters map[string]any) (*Driver, error) {
	// Providing no values for these is valid in case the user is authenticating
	// with an IAM on an ec2 instance (in which case the instance credentials will
	// be summoned when GetAuth is called).
	accessKey := parameters["accesskey"]
	if accessKey == nil {
		accessKey = ""
	}
	secretKey := parameters["secretkey"]
	if secretKey == nil {
		secretKey = ""
	}

	regionEndpoint := parameters["regionendpoint"]
	if regionEndpoint == nil {
		regionEndpoint = ""
	}

	forcePathStyleBool := true
	forcePathStyle := parameters["forcepathstyle"]
	switch forcePathStyle := forcePathStyle.(type) {
	case string:
		b, err := strconv.ParseBool(forcePathStyle)
		if err != nil {
			return nil, fmt.Errorf("the forcePathStyle parameter should be a boolean")
		}
		forcePathStyleBool = b
	case bool:
		forcePathStyleBool = forcePathStyle
	case nil:
		// do nothing
	default:
		return nil, fmt.Errorf("the forcePathStyle parameter should be a boolean")
	}

	regionName := parameters["region"]
	if regionName == nil || fmt.Sprint(regionName) == "" {
		return nil, fmt.Errorf("no region parameter provided")
	}
	region := fmt.Sprint(regionName)
	// Don't check the region value if a custom endpoint is provided.
	if regionEndpoint == "" {
		if _, ok := validRegions[region]; !ok {
			return nil, fmt.Errorf("invalid region provided: %v", region)
		}
	}

	bucket := parameters["bucket"]
	if bucket == nil || fmt.Sprint(bucket) == "" {
		return nil, fmt.Errorf("no bucket parameter provided")
	}

	encryptBool := false
	encrypt := parameters["encrypt"]
	switch encrypt := encrypt.(type) {
	case string:
		b, err := strconv.ParseBool(encrypt)
		if err != nil {
			return nil, fmt.Errorf("the encrypt parameter should be a boolean")
		}
		encryptBool = b
	case bool:
		encryptBool = encrypt
	case nil:
		// do nothing
	default:
		return nil, fmt.Errorf("the encrypt parameter should be a boolean")
	}

	secureBool := true
	secure := parameters["secure"]
	switch secure := secure.(type) {
	case string:
		b, err := strconv.ParseBool(secure)
		if err != nil {
			return nil, fmt.Errorf("the secure parameter should be a boolean")
		}
		secureBool = b
	case bool:
		secureBool = secure
	case nil:
		// do nothing
	default:
		return nil, fmt.Errorf("the secure parameter should be a boolean")
	}

	skipVerifyBool := false
	skipVerify := parameters["skipverify"]
	switch skipVerify := skipVerify.(type) {
	case string:
		b, err := strconv.ParseBool(skipVerify)
		if err != nil {
			return nil, fmt.Errorf("the skipVerify parameter should be a boolean")
		}
		skipVerifyBool = b
	case bool:
		skipVerifyBool = skipVerify
	case nil:
		// do nothing
	default:
		return nil, fmt.Errorf("the skipVerify parameter should be a boolean")
	}

	v4Bool := true
	v4auth := parameters["v4auth"]
	switch v4auth := v4auth.(type) {
	case string:
		b, err := strconv.ParseBool(v4auth)
		if err != nil {
			return nil, fmt.Errorf("the v4auth parameter should be a boolean")
		}
		v4Bool = b
	case bool:
		v4Bool = v4auth
	case nil:
		// do nothing
	default:
		return nil, fmt.Errorf("the v4auth parameter should be a boolean")
	}

	keyID := parameters["keyid"]
	if keyID == nil {
		keyID = ""
	}

	chunkSize, err := getParameterAsInt64(
		parameters, "chunksize",
		defaultChunkSize, minChunkSize, maxChunkSize,
	)
	if err != nil {
		return nil, err
	}

	multipartCopyChunkSize, err := getParameterAsInt64(
		parameters,
		"multipartcopychunksize",
		defaultMultipartCopyChunkSize,
		minChunkSize,
		maxChunkSize,
	)
	if err != nil {
		return nil, err
	}

	multipartCopyMaxConcurrency, err := getParameterAsInt64(
		parameters,
		"multipartcopymaxconcurrency",
		defaultMultipartCopyMaxConcurrency,
		1,
		math.MaxInt64,
	)
	if err != nil {
		return nil, err
	}

	multipartCopyThresholdSize, err := getParameterAsInt64(
		parameters,
		"multipartcopythresholdsize",
		defaultMultipartCopyThresholdSize,
		0,
		maxChunkSize,
	)
	if err != nil {
		return nil, err
	}

	rootDirectory := parameters["rootdirectory"]
	if rootDirectory == nil {
		rootDirectory = ""
	}

	storageClass := s3.StorageClassStandard
	storageClassParam := parameters["storageclass"]
	if storageClassParam != nil {
		storageClassString, ok := storageClassParam.(string)
		if !ok {
			return nil, fmt.Errorf(
				"the storageclass parameter must be one of %v, %v invalid",
				s3StorageClasses,
				storageClassParam,
			)
		}
		// All valid storage class parameters are UPPERCASE, so be a bit more flexible here
		storageClassString = strings.ToUpper(storageClassString)
		if storageClassString != noStorageClass &&
			storageClassString != s3.StorageClassStandard &&
			storageClassString != s3.StorageClassReducedRedundancy &&
			storageClassString != s3.StorageClassStandardIa &&
			storageClassString != s3.StorageClassOnezoneIa &&
			storageClassString != s3.StorageClassIntelligentTiering &&
			storageClassString != s3.StorageClassOutposts &&
			storageClassString != s3.StorageClassGlacierIr {
			return nil, fmt.Errorf(
				"the storageclass parameter must be one of %v, %v invalid",
				s3StorageClasses,
				storageClassParam,
			)
		}
		storageClass = storageClassString
	}

	userAgent := parameters["useragent"]
	if userAgent == nil {
		userAgent = ""
	}

	objectACL := s3.ObjectCannedACLPrivate
	objectACLParam := parameters["objectacl"]
	if objectACLParam != nil {
		objectACLString, ok := objectACLParam.(string)
		if !ok {
			return nil, fmt.Errorf(
				"invalid value for objectacl parameter: %v",
				objectACLParam,
			)
		}

		if _, ok = validObjectACLs[objectACLString]; !ok {
			return nil, fmt.Errorf(
				"invalid value for objectacl parameter: %v",
				objectACLParam,
			)
		}
		objectACL = objectACLString
	}

	useDualStackBool := false
	useDualStack := parameters["usedualstack"]
	switch useDualStack := useDualStack.(type) {
	case string:
		b, err := strconv.ParseBool(useDualStack)
		if err != nil {
			return nil, fmt.Errorf("the useDualStack parameter should be a boolean")
		}
		useDualStackBool = b
	case bool:
		useDualStackBool = useDualStack
	case nil:
		// do nothing
	default:
		return nil, fmt.Errorf("the useDualStack parameter should be a boolean")
	}

	sessionToken := ""

	accelerateBool := false
	accelerate := parameters["accelerate"]
	switch accelerate := accelerate.(type) {
	case string:
		b, err := strconv.ParseBool(accelerate)
		if err != nil {
			return nil, fmt.Errorf("the accelerate parameter should be a boolean")
		}
		accelerateBool = b
	case bool:
		accelerateBool = accelerate
	case nil:
		// do nothing
	default:
		return nil, fmt.Errorf("the accelerate parameter should be a boolean")
	}

	params := DriverParameters{
		fmt.Sprint(accessKey),
		fmt.Sprint(secretKey),
		fmt.Sprint(bucket),
		region,
		fmt.Sprint(regionEndpoint),
		forcePathStyleBool,
		encryptBool,
		fmt.Sprint(keyID),
		secureBool,
		skipVerifyBool,
		v4Bool,
		chunkSize,
		multipartCopyChunkSize,
		multipartCopyMaxConcurrency,
		multipartCopyThresholdSize,
		fmt.Sprint(rootDirectory),
		storageClass,
		fmt.Sprint(userAgent),
		objectACL,
		fmt.Sprint(sessionToken),
		useDualStackBool,
		accelerateBool,
		getS3LogLevelFromParam(parameters["loglevel"]), //nolint:contextcheck
	}

	return New(ctx, params)
}

func getS3LogLevelFromParam(param any) aws.LogLevelType {
	if param == nil {
		return aws.LogOff
	}
	logLevelParam, ok := param.(string)
	if !ok {
		log.Ctx(context.Background()).Warn().Msg("Error: param is not of type string")
	}
	var logLevel aws.LogLevelType
	switch strings.ToLower(logLevelParam) {
	case "off":
		logLevel = aws.LogOff
	case "debug":
		logLevel = aws.LogDebug
	case "debugwithsigning":
		logLevel = aws.LogDebugWithSigning
	case "debugwithhttpbody":
		logLevel = aws.LogDebugWithHTTPBody
	case "debugwithrequestretries":
		logLevel = aws.LogDebugWithRequestRetries
	case "debugwithrequesterrors":
		logLevel = aws.LogDebugWithRequestErrors
	case "debugwitheventstreambody":
		logLevel = aws.LogDebugWithEventStreamBody
	default:
		logLevel = aws.LogOff
	}
	return logLevel
}

// getParameterAsInt64 converts parameters[name] to an int64 value (using
// defaultt if nil), verifies it is no smaller than min, and returns it.
func getParameterAsInt64(
	parameters map[string]any,
	name string,
	defaultt int64,
	minSize int64,
	maxSize int64,
) (int64, error) {
	rv := defaultt
	param := parameters[name]
	switch v := param.(type) {
	case string:
		vv, err := strconv.ParseInt(v, 0, 64)
		if err != nil {
			return 0, fmt.Errorf("%s parameter must be an integer, %v invalid", name, param)
		}
		rv = vv
	case int64:
		rv = v
	case int, uint, int32, uint32, uint64:
		rv = reflect.ValueOf(v).Convert(reflect.TypeOf(rv)).Int()
	case nil:
		// do nothing
	default:
		return 0, fmt.Errorf("invalid value for %s: %#v", name, param)
	}

	if rv < minSize || rv > maxSize {
		return 0, fmt.Errorf(
			"the %s %#v parameter should be a number between %d and %d (inclusive)",
			name,
			rv,
			minSize,
			maxSize,
		)
	}

	return rv, nil
}

// New constructs a new Driver with the given AWS credentials, region, encryption flag, and
// bucketName.
func New(_ context.Context, params DriverParameters) (*Driver, error) {
	if !params.V4Auth &&
		(params.RegionEndpoint == "" ||
			strings.Contains(params.RegionEndpoint, "s3.amazonaws.com")) {
		return nil, fmt.Errorf("on Amazon S3 this storage driver can only be used with v4 authentication")
	}

	awsConfig := aws.NewConfig().WithLogLevel(params.LogLevel)

	if params.AccessKey != "" && params.SecretKey != "" {
		creds := credentials.NewStaticCredentials(
			params.AccessKey,
			params.SecretKey,
			params.SessionToken,
		)
		awsConfig.WithCredentials(creds)
	}

	if params.RegionEndpoint != "" {
		awsConfig.WithEndpoint(params.RegionEndpoint)
		awsConfig.WithS3ForcePathStyle(params.ForcePathStyle)
	}

	awsConfig.WithS3UseAccelerate(params.Accelerate)
	awsConfig.WithRegion(params.Region)
	awsConfig.WithDisableSSL(!params.Secure)
	if params.UseDualStack {
		awsConfig.UseDualStackEndpoint = endpoints.DualStackEndpointStateEnabled
	}

	if params.SkipVerify {
		httpTransport, ok := http.DefaultTransport.(*http.Transport)
		if !ok {
			return nil, fmt.Errorf("failed to get default transport")
		}
		httpTransport = httpTransport.Clone()
		httpTransport.TLSClientConfig = &tls.Config{InsecureSkipVerify: true, MinVersion: tls.VersionTLS12}
		awsConfig.WithHTTPClient(
			&http.Client{
				Transport: httpTransport,
			},
		)
	}

	sess, err := session.NewSession(awsConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to create new session with aws config: %w", err)
	}

	if params.UserAgent != "" {
		sess.Handlers.Build.PushBack(request.MakeAddToUserAgentFreeFormHandler(params.UserAgent))
	}

	s3obj := s3.New(sess)

	// enable S3 compatible signature v2 signing instead
	if !params.V4Auth {
		setv2Handlers(s3obj)
	}

	d := &driver{
		S3:                          s3obj,
		Bucket:                      params.Bucket,
		ChunkSize:                   params.ChunkSize,
		Encrypt:                     params.Encrypt,
		KeyID:                       params.KeyID,
		MultipartCopyChunkSize:      params.MultipartCopyChunkSize,
		MultipartCopyMaxConcurrency: params.MultipartCopyMaxConcurrency,
		MultipartCopyThresholdSize:  params.MultipartCopyThresholdSize,
		RootDirectory:               params.RootDirectory,
		StorageClass:                params.StorageClass,
		ObjectACL:                   params.ObjectACL,
		pool: &sync.Pool{
			New: func() any {
				return &buffer{
					data: make([]byte, 0, params.ChunkSize),
				}
			},
		},
	}

	return &Driver{
		baseEmbed: baseEmbed{
			Base: base.Base{
				StorageDriver: d,
			},
		},
	}, nil
}

// Implement the storagedriver.StorageDriver interface

func (d *driver) Name() string {
	return driverName
}

// GetContent retrieves the content stored at "path" as a []byte.
func (d *driver) GetContent(ctx context.Context, path string) ([]byte, error) {
	reader, err := d.Reader(ctx, path, 0)
	if err != nil {
		return nil, err
	}
	return io.ReadAll(reader)
}

// PutContent stores the []byte content at a location designated by "path".
func (d *driver) PutContent(ctx context.Context, path string, contents []byte) error {
	log.Ctx(ctx).Trace().Msgf("[AWS] PutContent: %s", path)
	_, err := d.S3.PutObjectWithContext(
		ctx, &s3.PutObjectInput{
			Bucket:               aws.String(d.Bucket),
			Key:                  aws.String(d.s3Path(path)),
			ContentType:          d.getContentType(),
			ACL:                  d.getACL(),
			ServerSideEncryption: d.getEncryptionMode(),
			SSEKMSKeyId:          d.getSSEKMSKeyID(),
			StorageClass:         d.getStorageClass(),
			Body:                 bytes.NewReader(contents),
		},
	)
	return parseError(path, err)
}

// Reader retrieves an io.ReadCloser for the content stored at "path" with a
// given byte offset.
func (d *driver) Reader(ctx context.Context, path string, offset int64) (io.ReadCloser, error) {
	log.Ctx(ctx).Trace().Msgf("[AWS] GetObject: %s", path)
	resp, err := d.S3.GetObjectWithContext(
		ctx, &s3.GetObjectInput{
			Bucket: aws.String(d.Bucket),
			Key:    aws.String(d.s3Path(path)),
			Range:  aws.String("bytes=" + strconv.FormatInt(offset, 10) + "-"),
		},
	)
	if err != nil {
		var s3Err awserr.Error
		if ok := errors.As(err, &s3Err); ok && s3Err.Code() == "InvalidRange" {
			return io.NopCloser(bytes.NewReader(nil)), nil
		}

		return nil, parseError(path, err)
	}
	return resp.Body, nil
}

// Writer returns a FileWriter which will store the content written to it
// at the location designated by "path" after the call to Commit.
// It only allows appending to paths with zero size committed content,
// in which the existing content is overridden with the new content.
// It returns storagedriver.Error when appending to paths
// with non-zero committed content.
func (d *driver) Writer(ctx context.Context, path string, appendMode bool) (storagedriver.FileWriter, error) {
	key := d.s3Path(path)
	if !appendMode {
		log.Ctx(ctx).Trace().Msgf("[AWS] CreateMultipartUpload: %s", path)
		resp, err := d.S3.CreateMultipartUploadWithContext(
			ctx, &s3.CreateMultipartUploadInput{
				Bucket:               aws.String(d.Bucket),
				Key:                  aws.String(key),
				ContentType:          d.getContentType(),
				ACL:                  d.getACL(),
				ServerSideEncryption: d.getEncryptionMode(),
				SSEKMSKeyId:          d.getSSEKMSKeyID(),
				StorageClass:         d.getStorageClass(),
			},
		)
		if err != nil {
			return nil, err
		}
		return d.newWriter(ctx, key, *resp.UploadId, nil), nil
	}

	listMultipartUploadsInput := &s3.ListMultipartUploadsInput{
		Bucket: aws.String(d.Bucket),
		Prefix: aws.String(key),
	}
	for {
		log.Ctx(ctx).Trace().Msgf("[AWS] ListMultipartUploads: %s", path)
		resp, err := d.S3.ListMultipartUploadsWithContext(ctx, listMultipartUploadsInput)
		if err != nil {
			return nil, parseError(path, err)
		}

		// resp.Uploads can only be empty on the first call
		// if there were no more results to return after the first call, resp.IsTruncated would have been false
		// and the loop would be exited without recalling ListMultipartUploads
		if len(resp.Uploads) == 0 {
			fi, err := d.Stat(ctx, path)
			if err != nil {
				return nil, parseError(path, err)
			}

			if fi.Size() == 0 {
				log.Ctx(ctx).Trace().Msgf("[AWS] CreateMultipartUpload: %s", path)
				resp, err := d.S3.CreateMultipartUploadWithContext(
					ctx, &s3.CreateMultipartUploadInput{
						Bucket:               aws.String(d.Bucket),
						Key:                  aws.String(key),
						ContentType:          d.getContentType(),
						ACL:                  d.getACL(),
						ServerSideEncryption: d.getEncryptionMode(),
						SSEKMSKeyId:          d.getSSEKMSKeyID(),
						StorageClass:         d.getStorageClass(),
					},
				)
				if err != nil {
					return nil, err
				}
				return d.newWriter(ctx, key, *resp.UploadId, nil), nil
			}
			return nil, storagedriver.Error{
				DriverName: driverName,
				Detail:     fmt.Errorf("append to zero-size path %s unsupported", path),
			}
		}

		var allParts []*s3.Part
		for _, multi := range resp.Uploads {
			if key != *multi.Key {
				continue
			}

			log.Ctx(ctx).Trace().Msgf("[AWS] ListParts: %s", path)
			partsList, err := d.S3.ListPartsWithContext(
				ctx, &s3.ListPartsInput{
					Bucket:   aws.String(d.Bucket),
					Key:      aws.String(key),
					UploadId: multi.UploadId,
				},
			)
			if err != nil {
				return nil, parseError(path, err)
			}
			allParts = append(allParts, partsList.Parts...)
			for *partsList.IsTruncated {
				log.Ctx(ctx).Trace().Msgf("[AWS] ListParts: %s", path)
				partsList, err = d.S3.ListPartsWithContext(
					ctx, &s3.ListPartsInput{
						Bucket:           aws.String(d.Bucket),
						Key:              aws.String(key),
						UploadId:         multi.UploadId,
						PartNumberMarker: partsList.NextPartNumberMarker,
					},
				)
				if err != nil {
					return nil, parseError(path, err)
				}
				allParts = append(allParts, partsList.Parts...)
			}
			return d.newWriter(ctx, key, *multi.UploadId, allParts), nil
		}

		// resp.NextUploadIdMarker must have at least one element or we would have returned not found
		listMultipartUploadsInput.UploadIdMarker = resp.NextUploadIdMarker

		// from the s3 api docs, IsTruncated "specifies whether (true) or not (false) all of the results were returned"
		// if everything has been returned, break
		if resp.IsTruncated == nil || !*resp.IsTruncated {
			break
		}
	}
	return nil, storagedriver.PathNotFoundError{Path: path}
}

// Stat retrieves the FileInfo for the given path, including the current size
// in bytes and the creation time.
func (d *driver) Stat(ctx context.Context, path string) (storagedriver.FileInfo, error) {
	log.Ctx(ctx).Trace().Msgf("[AWS] ListObjectsV2: %s", path)
	resp, err := d.S3.ListObjectsV2WithContext(
		ctx, &s3.ListObjectsV2Input{
			Bucket:  aws.String(d.Bucket),
			Prefix:  aws.String(d.s3Path(path)),
			MaxKeys: aws.Int64(1),
		},
	)
	if err != nil {
		return nil, err
	}

	fi := storagedriver.FileInfoFields{
		Path: path,
	}

	switch {
	case len(resp.Contents) == 1:
		if *resp.Contents[0].Key != d.s3Path(path) {
			fi.IsDir = true
		} else {
			fi.IsDir = false
			fi.Size = *resp.Contents[0].Size
			fi.ModTime = *resp.Contents[0].LastModified
		}
	case len(resp.CommonPrefixes) == 1:
		fi.IsDir = true
	default:
		return nil, storagedriver.PathNotFoundError{Path: path}
	}

	return storagedriver.FileInfoInternal{FileInfoFields: fi}, nil
}

// List returns a list of the objects that are direct descendants of the given path.
func (d *driver) List(ctx context.Context, opath string) ([]string, error) {
	path := opath
	if path != "/" && path[len(path)-1] != '/' {
		path += "/"
	}

	// This is to cover for the cases when the rootDirectory of the driver is either "" or "/".
	// In those cases, there is no root prefix to replace and we must actually add a "/" to all
	// results in order to keep them as valid paths as recognized by storagedriver.PathRegexp
	prefix := ""
	if d.s3Path("") == "" {
		prefix = "/"
	}

	log.Ctx(ctx).Trace().Msgf("[AWS] ListObjectsV2: %s", path)
	resp, err := d.S3.ListObjectsV2WithContext(
		ctx, &s3.ListObjectsV2Input{
			Bucket:    aws.String(d.Bucket),
			Prefix:    aws.String(d.s3Path(path)),
			Delimiter: aws.String("/"),
			MaxKeys:   aws.Int64(listMax),
		},
	)
	if err != nil {
		return nil, parseError(opath, err)
	}

	files := []string{}
	directories := []string{}

	for {
		for _, key := range resp.Contents {
			files = append(files, strings.Replace(*key.Key, d.s3Path(""), prefix, 1))
		}

		for _, commonPrefix := range resp.CommonPrefixes {
			commonPrefix := *commonPrefix.Prefix
			directories = append(
				directories,
				strings.Replace(commonPrefix[0:len(commonPrefix)-1], d.s3Path(""), prefix, 1),
			)
		}

		if *resp.IsTruncated {
			log.Ctx(ctx).Trace().Msgf("[AWS] ListObjectsV2: %s", path)
			resp, err = d.S3.ListObjectsV2WithContext(
				ctx, &s3.ListObjectsV2Input{
					Bucket:            aws.String(d.Bucket),
					Prefix:            aws.String(d.s3Path(path)),
					Delimiter:         aws.String("/"),
					MaxKeys:           aws.Int64(listMax),
					ContinuationToken: resp.NextContinuationToken,
				},
			)
			if err != nil {
				return nil, err
			}
		} else {
			break
		}
	}

	if opath != "/" {
		if len(files) == 0 && len(directories) == 0 {
			// Treat empty response as missing directory, since we don't actually
			// have directories in s3.
			return nil, storagedriver.PathNotFoundError{Path: opath}
		}
	}

	return append(files, directories...), nil
}

// Move moves an object stored at sourcePath to destPath, removing the original
// object.
func (d *driver) Move(ctx context.Context, sourcePath string, destPath string) error {
	/* This is terrible, but aws doesn't have an actual move. */
	if err := d.copy(ctx, sourcePath, destPath); err != nil {
		return err
	}
	return d.Delete(ctx, sourcePath)
}

// copy copies an object stored at sourcePath to destPath.
func (d *driver) copy(ctx context.Context, sourcePath string, destPath string) error {
	// S3 can copy objects up to 5 GB in size with a single PUT Object - Copy
	// operation. For larger objects, the multipart upload API must be used.
	//
	// Empirically, multipart copy is fastest with 32 MB parts and is faster
	// than PUT Object - Copy for objects larger than 32 MB.

	fileInfo, err := d.Stat(ctx, sourcePath)
	if err != nil {
		return parseError(sourcePath, err)
	}

	if fileInfo.Size() <= d.MultipartCopyThresholdSize {
		log.Ctx(ctx).Trace().Msgf("[AWS] CopyObject: %s -> %s", sourcePath, destPath)
		_, err := d.S3.CopyObjectWithContext(
			ctx, &s3.CopyObjectInput{
				Bucket:               aws.String(d.Bucket),
				Key:                  aws.String(d.s3Path(destPath)),
				ContentType:          d.getContentType(),
				ACL:                  d.getACL(),
				ServerSideEncryption: d.getEncryptionMode(),
				SSEKMSKeyId:          d.getSSEKMSKeyID(),
				StorageClass:         d.getStorageClass(),
				CopySource:           aws.String(d.Bucket + "/" + d.s3Path(sourcePath)),
			},
		)
		if err != nil {
			return parseError(sourcePath, err)
		}
		return nil
	}

	log.Ctx(ctx).Trace().Msgf("[AWS] CreateMultipartUpload: %s", destPath)
	createResp, err := d.S3.CreateMultipartUploadWithContext(
		ctx, &s3.CreateMultipartUploadInput{
			Bucket:               aws.String(d.Bucket),
			Key:                  aws.String(d.s3Path(destPath)),
			ContentType:          d.getContentType(),
			ACL:                  d.getACL(),
			SSEKMSKeyId:          d.getSSEKMSKeyID(),
			ServerSideEncryption: d.getEncryptionMode(),
			StorageClass:         d.getStorageClass(),
		},
	)
	if err != nil {
		return err
	}

	numParts := (fileInfo.Size() + d.MultipartCopyChunkSize - 1) / d.MultipartCopyChunkSize
	completedParts := make([]*s3.CompletedPart, numParts)
	errChan := make(chan error, numParts)
	limiter := make(chan struct{}, d.MultipartCopyMaxConcurrency)

	for i := range completedParts {
		i := int64(i)
		go func() {
			limiter <- struct{}{}
			firstByte := i * d.MultipartCopyChunkSize
			lastByte := firstByte + d.MultipartCopyChunkSize - 1
			if lastByte >= fileInfo.Size() {
				lastByte = fileInfo.Size() - 1
			}
			log.Ctx(ctx).Trace().Msgf("[AWS] [%d] UploadPartCopy: %s -> %s", i, sourcePath, destPath)
			uploadResp, err := d.S3.UploadPartCopyWithContext(
				ctx, &s3.UploadPartCopyInput{
					Bucket:          aws.String(d.Bucket),
					CopySource:      aws.String(d.Bucket + "/" + d.s3Path(sourcePath)),
					Key:             aws.String(d.s3Path(destPath)),
					PartNumber:      aws.Int64(i + 1),
					UploadId:        createResp.UploadId,
					CopySourceRange: aws.String(fmt.Sprintf("bytes=%d-%d", firstByte, lastByte)),
				},
			)
			if err == nil {
				completedParts[i] = &s3.CompletedPart{
					ETag:       uploadResp.CopyPartResult.ETag,
					PartNumber: aws.Int64(i + 1),
				}
			}
			errChan <- err
			<-limiter
		}()
	}

	for range completedParts {
		err := <-errChan
		if err != nil {
			return err
		}
	}

	log.Ctx(ctx).Trace().Msgf("[AWS] CompleteMultipartUpload: %s", destPath)
	_, err = d.S3.CompleteMultipartUploadWithContext(
		ctx, &s3.CompleteMultipartUploadInput{
			Bucket:          aws.String(d.Bucket),
			Key:             aws.String(d.s3Path(destPath)),
			UploadId:        createResp.UploadId,
			MultipartUpload: &s3.CompletedMultipartUpload{Parts: completedParts},
		},
	)
	return err
}

// Delete recursively deletes all objects stored at "path" and its subpaths.
// We must be careful since S3 does not guarantee read after delete consistency.
func (d *driver) Delete(ctx context.Context, path string) error {
	s3Objects := make([]*s3.ObjectIdentifier, 0, listMax)
	s3Path := d.s3Path(path)
	listObjectsInput := &s3.ListObjectsV2Input{
		Bucket: aws.String(d.Bucket),
		Prefix: aws.String(s3Path),
	}

	for {
		// list all the objects
		log.Ctx(ctx).Trace().Msgf("[AWS] List all the objects: %s", path)
		resp, err := d.S3.ListObjectsV2WithContext(ctx, listObjectsInput)

		// resp.Contents can only be empty on the first call
		// if there were no more results to return after the first call, resp.IsTruncated would have been false
		// and the loop would exit without recalling ListObjects
		if err != nil || len(resp.Contents) == 0 {
			return storagedriver.PathNotFoundError{Path: path}
		}

		for _, key := range resp.Contents {
			// Skip if we encounter a key that is not a subpath (so that deleting "/a" does not delete "/ab").
			if len(*key.Key) > len(s3Path) && (*key.Key)[len(s3Path)] != '/' {
				continue
			}
			s3Objects = append(
				s3Objects, &s3.ObjectIdentifier{
					Key: key.Key,
				},
			)
		}

		// Delete objects only if the list is not empty, otherwise S3 API returns a cryptic error
		if len(s3Objects) > 0 {
			// NOTE: according to AWS docs
			// https://docs.aws.amazon.com/AmazonS3/latest/API/API_ListObjectsV2.html
			// by default the response returns up to 1,000 key names. The response _might_
			// contain fewer keys but it will never contain more.
			// 10000 keys is coincidentally (?) also the max number of keys that can be
			// deleted in a single Delete operation, so we'll just smack
			// Delete here straight away and reset the object slice when successful.
			log.Ctx(ctx).Trace().Msgf("[AWS] DeleteObjects: %s", path)
			resp, err := d.S3.DeleteObjectsWithContext(
				ctx, &s3.DeleteObjectsInput{
					Bucket: aws.String(d.Bucket),
					Delete: &s3.Delete{
						Objects: s3Objects,
						Quiet:   aws.Bool(false),
					},
				},
			)
			if err != nil {
				return err
			}

			if len(resp.Errors) > 0 {
				// NOTE: AWS SDK s3.Error does not implement error interface which
				// is pretty intensely sad, so we have to do away with this for now.
				errs := make([]error, 0, len(resp.Errors))
				for _, err := range resp.Errors {
					errs = append(errs, errors.New(err.String()))
				}
				return storagedriver.StorageDriverError{
					DriverName: driverName,
					Errs:       errs,
				}
			}
		}
		// NOTE: we don't want to reallocate
		// the slice so we simply "reset" it
		s3Objects = s3Objects[:0]

		// resp.Contents must have at least one element or we would have returned not found
		listObjectsInput.StartAfter = resp.Contents[len(resp.Contents)-1].Key

		// from the s3 api docs, IsTruncated "specifies whether (true) or not (false) all of the results were returned"
		// if everything has been returned, break
		if resp.IsTruncated == nil || !*resp.IsTruncated {
			break
		}
	}

	return nil
}

// RedirectURL returns a URL which may be used to retrieve the content stored at the given path.
func (d *driver) RedirectURL(ctx context.Context, method string, path string, filename string) (string, error) {
	expiresIn := 20 * time.Minute

	var req *request.Request

	switch method {
	case http.MethodGet:
		input := &s3.GetObjectInput{
			Bucket: aws.String(d.Bucket),
			Key:    aws.String(d.s3Path(path)),
		}
		if filename != "" {
			input.ResponseContentDisposition = aws.String(fmt.Sprintf("attachment; filename=\"%s\"", filename))
		}
		req, _ = d.S3.GetObjectRequest(input)
	case http.MethodHead:
		req, _ = d.S3.HeadObjectRequest(
			&s3.HeadObjectInput{
				Bucket: aws.String(d.Bucket),
				Key:    aws.String(d.s3Path(path)),
			},
		)
	default:
		return "", nil
	}

	log.Ctx(ctx).Debug().Msgf("[AWS] Generating presigned URL for %s %s", method, path)
	return req.Presign(expiresIn)
}

// Walk traverses a filesystem defined within driver, starting
// from the given path, calling f on each file.
func (d *driver) Walk(
	ctx context.Context,
	from string,
	f storagedriver.WalkFn,
	options ...func(*storagedriver.WalkOptions),
) error {
	walkOptions := &storagedriver.WalkOptions{}
	for _, o := range options {
		o(walkOptions)
	}

	var objectCount int64
	if err := d.doWalk(ctx, &objectCount, from, walkOptions.StartAfterHint, f); err != nil {
		return err
	}

	return nil
}

func (d *driver) doWalk(
	parentCtx context.Context,
	objectCount *int64,
	from string,
	startAfter string,
	f storagedriver.WalkFn,
) error {
	var (
		retError error
		// the most recent directory walked for de-duping
		prevDir string
		// the most recent skip directory to avoid walking over undesirable files
		prevSkipDir string
	)
	prevDir = from

	path := from
	if !strings.HasSuffix(path, "/") {
		path += "/"
	}

	prefix := ""
	if d.s3Path("") == "" {
		prefix = "/"
	}

	listObjectsInput := &s3.ListObjectsV2Input{
		Bucket:     aws.String(d.Bucket),
		Prefix:     aws.String(d.s3Path(path)),
		MaxKeys:    aws.Int64(listMax),
		StartAfter: aws.String(d.s3Path(startAfter)),
	}

	ctx, done := dcontext.WithTrace(parentCtx)
	defer done("s3aws.ListObjectsV2PagesWithContext(%s)", listObjectsInput)

	// When the "delimiter" argument is omitted, the S3 list
	// API will list all objects in the bucket
	// recursively, omitting directory paths.
	// Objects are listed in sorted, depth-first order so we
	// can infer all the directories by comparing each object
	// path to the last one we saw. See:
	// https://docs.aws.amazon.com/AmazonS3/latest/userguide/ListingKeysUsingAPIs.html

	// With files returned in sorted depth-first order, directories
	// are inferred in the same order.
	// ErrSkipDir is handled by explicitly skipping over any files
	//  under the skipped directory. This may be sub-optimal
	// for extreme edge cases but for the general use case
	// in a registry, this is orders of magnitude
	// faster than a more explicit recursive implementation.
	log.Ctx(ctx).Trace().Msgf("[AWS] Listing objects in %s", path)
	listObjectErr := d.S3.ListObjectsV2PagesWithContext(
		ctx,
		listObjectsInput,
		func(objects *s3.ListObjectsV2Output, _ bool) bool {
			walkInfos := make([]storagedriver.FileInfoInternal, 0, len(objects.Contents))

			for _, file := range objects.Contents {
				filePath := strings.Replace(*file.Key, d.s3Path(""), prefix, 1)

				// get a list of all inferred directories between the previous directory and this file
				dirs := directoryDiff(prevDir, filePath)
				if len(dirs) > 0 {
					for _, dir := range dirs {
						walkInfos = append(
							walkInfos, storagedriver.FileInfoInternal{
								FileInfoFields: storagedriver.FileInfoFields{
									IsDir: true,
									Path:  dir,
								},
							},
						)
						prevDir = dir
					}
				}

				walkInfos = append(
					walkInfos, storagedriver.FileInfoInternal{
						FileInfoFields: storagedriver.FileInfoFields{
							IsDir:   false,
							Size:    *file.Size,
							ModTime: *file.LastModified,
							Path:    filePath,
						},
					},
				)
			}

			for _, walkInfo := range walkInfos {
				// skip any results under the last skip directory
				if prevSkipDir != "" && strings.HasPrefix(walkInfo.Path(), prevSkipDir) {
					continue
				}

				err := f(walkInfo)
				*objectCount++

				if err != nil {
					if errors.Is(err, storagedriver.ErrSkipDir) {
						prevSkipDir = walkInfo.Path()
						continue
					}
					if errors.Is(err, storagedriver.ErrFilledBuffer) {
						return false
					}
					retError = err
					return false
				}
			}
			return true
		},
	)

	if retError != nil {
		return retError
	}

	if listObjectErr != nil {
		return listObjectErr
	}

	return nil
}

// directoryDiff finds all directories that are not in common between
// the previous and current paths in sorted order.
//
// # Examples
//
//	directoryDiff("/path/to/folder", "/path/to/folder/folder/file")
//	// => [ "/path/to/folder/folder" ]
//
//	directoryDiff("/path/to/folder/folder1", "/path/to/folder/folder2/file")
//	// => [ "/path/to/folder/folder2" ]
//
//	directoryDiff("/path/to/folder/folder1/file", "/path/to/folder/folder2/file")
//	// => [ "/path/to/folder/folder2" ]
//
//	directoryDiff("/path/to/folder/folder1/file", "/path/to/folder/folder2/folder1/file")
//	// => [ "/path/to/folder/folder2", "/path/to/folder/folder2/folder1" ]
//
//	directoryDiff("/", "/path/to/folder/folder/file")
//	// => [ "/path", "/path/to", "/path/to/folder", "/path/to/folder/folder" ]
func directoryDiff(prev, current string) []string {
	var paths []string

	if prev == "" || current == "" {
		return paths
	}

	parent := current
	for {
		parent = filepath.Dir(parent)
		if parent == "/" || parent == prev || strings.HasPrefix(prev+"/", parent+"/") {
			break
		}
		paths = append(paths, parent)
	}
	reverse(paths)
	return paths
}

func reverse(s []string) {
	for i, j := 0, len(s)-1; i < j; i, j = i+1, j-1 {
		s[i], s[j] = s[j], s[i]
	}
}

func (d *driver) s3Path(path string) string {
	return strings.TrimLeft(strings.TrimRight(d.RootDirectory, "/")+path, "/")
}

// S3BucketKey returns the s3 bucket key for the given storage driver path.
func (d *Driver) S3BucketKey(path string) string {
	// We can ignore the error here as s3Path only returns error for empty paths
	// which is handled by the caller
	//nolint:errcheck
	return d.StorageDriver.(*driver).s3Path(path)
}

func parseError(path string, err error) error {
	var s3Err awserr.Error
	if ok := errors.As(err, &s3Err); ok && s3Err.Code() == "NoSuchKey" {
		return storagedriver.PathNotFoundError{Path: path}
	}

	return err
}

func (d *driver) getEncryptionMode() *string {
	if !d.Encrypt {
		return nil
	}
	if d.KeyID == "" {
		return aws.String("AES256")
	}
	return aws.String("aws:kms")
}

func (d *driver) getSSEKMSKeyID() *string {
	if d.KeyID != "" {
		return aws.String(d.KeyID)
	}
	return nil
}

func (d *driver) getContentType() *string {
	return aws.String("application/octet-stream")
}

func (d *driver) getACL() *string {
	return aws.String(d.ObjectACL)
}

func (d *driver) getStorageClass() *string {
	if d.StorageClass == noStorageClass {
		return nil
	}
	return aws.String(d.StorageClass)
}

// buffer is a static size bytes buffer.
type buffer struct {
	data []byte
}

// NewBuffer returns a new bytes buffer from driver's memory pool.
// The size of the buffer is static and set to params.ChunkSize.
func (d *driver) NewBuffer() *buffer {
	buf, ok := d.pool.Get().(*buffer)
	if !ok {
		return nil
	}
	return buf
}

// ReadFrom reads as much data as it can fit in from r without growing its size.
// It returns the number of bytes successfully read from r or error.
func (b *buffer) ReadFrom(r io.Reader) (offset int64, err error) {
	for len(b.data) < cap(b.data) && err == nil {
		var n int
		n, err = r.Read(b.data[len(b.data):cap(b.data)])
		offset += int64(n)
		b.data = b.data[:len(b.data)+n]
	}
	if err == io.EOF {
		err = nil
	}
	return offset, err
}

// Cap returns the capacity of the buffer's underlying byte slice.
func (b *buffer) Cap() int {
	return cap(b.data)
}

// Len returns the length of the data in the buffer.
func (b *buffer) Len() int {
	return len(b.data)
}

// Clear the buffer data.
func (b *buffer) Clear() {
	b.data = b.data[:0]
}

// writer attempts to upload parts to S3 in a buffered fashion where the last
// part is at least as large as the chunksize, so the multipart upload could be
// cleanly resumed in the future. This is violated if Close is called after less
// than a full chunk is written.
type writer struct {
	ctx       context.Context
	driver    *driver
	key       string
	uploadID  string
	parts     []*s3.Part
	size      int64
	ready     *buffer
	pending   *buffer
	closed    bool
	committed bool
	cancelled bool
}

func (d *driver) newWriter(ctx context.Context, key, uploadID string, parts []*s3.Part) storagedriver.FileWriter {
	var size int64
	for _, part := range parts {
		size += *part.Size
	}
	return &writer{
		ctx:      ctx,
		driver:   d,
		key:      key,
		uploadID: uploadID,
		parts:    parts,
		size:     size,
		ready:    d.NewBuffer(),
		pending:  d.NewBuffer(),
	}
}

type completedParts []*s3.CompletedPart

func (a completedParts) Len() int           { return len(a) }
func (a completedParts) Swap(i, j int)      { a[i], a[j] = a[j], a[i] }
func (a completedParts) Less(i, j int) bool { return *a[i].PartNumber < *a[j].PartNumber }

//nolint:gocognit
func (w *writer) Write(p []byte) (int, error) {
	switch {
	case w.closed:
		return 0, fmt.Errorf("already closed")
	case w.committed:
		return 0, fmt.Errorf("already committed")
	case w.cancelled:
		return 0, fmt.Errorf("already cancelled")
	}

	// If the last written part is smaller than minChunkSize, we need to make a
	// new multipart upload :sadface:
	if len(w.parts) > 0 && int(*w.parts[len(w.parts)-1].Size) < minChunkSize {
		completedUploadedParts := make(completedParts, len(w.parts))
		for i, part := range w.parts {
			completedUploadedParts[i] = &s3.CompletedPart{
				ETag:       part.ETag,
				PartNumber: part.PartNumber,
			}
		}

		sort.Sort(completedUploadedParts)

		_, err := w.driver.S3.CompleteMultipartUploadWithContext(
			w.ctx, &s3.CompleteMultipartUploadInput{
				Bucket:   aws.String(w.driver.Bucket),
				Key:      aws.String(w.key),
				UploadId: aws.String(w.uploadID),
				MultipartUpload: &s3.CompletedMultipartUpload{
					Parts: completedUploadedParts,
				},
			},
		)
		if err != nil {
			if _, aErr := w.driver.S3.AbortMultipartUploadWithContext(
				w.ctx, &s3.AbortMultipartUploadInput{
					Bucket:   aws.String(w.driver.Bucket),
					Key:      aws.String(w.key),
					UploadId: aws.String(w.uploadID),
				},
			); aErr != nil {
				return 0, errors.Join(err, aErr)
			}
			return 0, err
		}

		resp, err := w.driver.S3.CreateMultipartUploadWithContext(
			w.ctx, &s3.CreateMultipartUploadInput{
				Bucket:               aws.String(w.driver.Bucket),
				Key:                  aws.String(w.key),
				ContentType:          w.driver.getContentType(),
				ACL:                  w.driver.getACL(),
				ServerSideEncryption: w.driver.getEncryptionMode(),
				StorageClass:         w.driver.getStorageClass(),
			},
		)
		if err != nil {
			return 0, err
		}
		w.uploadID = *resp.UploadId

		// If the entire written file is smaller than minChunkSize, we need to make
		// a new part from scratch :double sad face:
		if w.size < minChunkSize {
			resp, err := w.driver.S3.GetObjectWithContext(
				w.ctx, &s3.GetObjectInput{
					Bucket: aws.String(w.driver.Bucket),
					Key:    aws.String(w.key),
				},
			)
			if err != nil {
				return 0, err
			}
			defer resp.Body.Close()

			// reset uploaded parts
			w.parts = nil
			w.ready.Clear()

			n, err := w.ready.ReadFrom(resp.Body)
			if err != nil {
				return 0, err
			}
			if resp.ContentLength != nil && n < *resp.ContentLength {
				return 0, io.ErrShortBuffer
			}
		} else {
			// Otherwise we can use the old file as the new first part
			copyPartResp, err := w.driver.S3.UploadPartCopyWithContext(
				w.ctx, &s3.UploadPartCopyInput{
					Bucket:     aws.String(w.driver.Bucket),
					CopySource: aws.String(w.driver.Bucket + "/" + w.key),
					Key:        aws.String(w.key),
					PartNumber: aws.Int64(1),
					UploadId:   resp.UploadId,
				},
			)
			if err != nil {
				return 0, err
			}
			w.parts = []*s3.Part{
				{
					ETag:       copyPartResp.CopyPartResult.ETag,
					PartNumber: aws.Int64(1),
					Size:       aws.Int64(w.size),
				},
			}
		}
	}

	var n int

	defer func() { w.size += int64(n) }()

	reader := bytes.NewReader(p)

	for reader.Len() > 0 {
		// NOTE: we do some seemingly unsafe conversions
		// from int64 to int in this for loop. These are fine as the
		// offset returned from buffer.ReadFrom can only ever be
		// maxChunkSize large which fits in to int. The reason why
		// we return int64 is to play nice with Go interfaces where
		// the buffer implements io.ReaderFrom interface.

		// fill up the ready parts buffer
		offset, err := w.ready.ReadFrom(reader)
		n += int(offset)
		if err != nil {
			return n, err
		}

		// try filling up the pending parts buffer
		offset, err = w.pending.ReadFrom(reader)
		n += int(offset)
		if err != nil {
			return n, err
		}

		// we filled up pending buffer, flush
		if w.pending.Len() == w.pending.Cap() {
			if err := w.flush(); err != nil {
				return n, err
			}
		}
	}

	return n, nil
}

func (w *writer) Size() int64 {
	return w.size
}
func (w *writer) Close() error {
	if w.closed {
		return fmt.Errorf("already closed")
	}
	w.closed = true

	defer func() {
		w.ready.Clear()
		w.driver.pool.Put(w.ready)
		w.pending.Clear()
		w.driver.pool.Put(w.pending)
	}()

	return w.flush()
}

func (w *writer) Cancel(ctx context.Context) error {
	if w.closed {
		return fmt.Errorf("already closed")
	} else if w.committed {
		return fmt.Errorf("already committed")
	}
	w.cancelled = true
	log.Ctx(ctx).Trace().Msgf("[AWS] Abort multipart upload for %s", w.key)
	_, err := w.driver.S3.AbortMultipartUploadWithContext(
		ctx, &s3.AbortMultipartUploadInput{
			Bucket:   aws.String(w.driver.Bucket),
			Key:      aws.String(w.key),
			UploadId: aws.String(w.uploadID),
		},
	)
	return err
}

func (w *writer) Commit(ctx context.Context) error {
	switch {
	case w.closed:
		return fmt.Errorf("already closed")
	case w.committed:
		return fmt.Errorf("already committed")
	case w.cancelled:
		return fmt.Errorf("already cancelled")
	}

	err := w.flush()
	if err != nil {
		return err
	}

	w.committed = true

	completedUploadedParts := make(completedParts, len(w.parts))
	for i, part := range w.parts {
		completedUploadedParts[i] = &s3.CompletedPart{
			ETag:       part.ETag,
			PartNumber: part.PartNumber,
		}
	}

	// This is an edge case when we are trying to upload an empty file as part of
	// the MultiPart upload. We get a PUT with Content-Length: 0 and sad things happen.
	// The result is we are trying to Complete MultipartUpload with an empty list of
	// completedUploadedParts which will always lead to 400 being returned from S3
	// See: https://docs.aws.amazon.com/sdk-for-go/api/service/s3/#CompletedMultipartUpload
	// Solution: we upload the empty i.e. 0 byte part as a single part and then append it
	// to the completedUploadedParts slice used to complete the Multipart upload.
	if len(w.parts) == 0 {
		log.Ctx(ctx).Trace().Msgf("[AWS] Upload empty part for %s", w.key)
		//nolint:contextcheck
		resp, err := w.driver.S3.UploadPartWithContext(w.ctx, &s3.UploadPartInput{
			Bucket:     aws.String(w.driver.Bucket),
			Key:        aws.String(w.key),
			PartNumber: aws.Int64(1),
			UploadId:   aws.String(w.uploadID),
			Body:       bytes.NewReader(nil),
		},
		)
		if err != nil {
			return err
		}
		tmp := completedUploadedParts

		tmp = append(
			tmp, &s3.CompletedPart{
				ETag:       resp.ETag,
				PartNumber: aws.Int64(1),
			},
		)

		completedUploadedParts = tmp
	}

	sort.Sort(completedUploadedParts)

	log.Ctx(ctx).Trace().Msgf("[AWS] Complete multipart upload for %s", w.key)
	//nolint:contextcheck
	_, err = w.driver.S3.CompleteMultipartUploadWithContext(w.ctx, &s3.CompleteMultipartUploadInput{
		Bucket:   aws.String(w.driver.Bucket),
		Key:      aws.String(w.key),
		UploadId: aws.String(w.uploadID),
		MultipartUpload: &s3.CompletedMultipartUpload{
			Parts: completedUploadedParts,
		},
	},
	)
	if err != nil {
		log.Ctx(ctx).Trace().Msgf("[AWS] Abort multipart upload for %s: %v", w.key, err)
		//nolint:contextcheck
		if _, aErr := w.driver.S3.AbortMultipartUploadWithContext(
			w.ctx, &s3.AbortMultipartUploadInput{
				Bucket:   aws.String(w.driver.Bucket),
				Key:      aws.String(w.key),
				UploadId: aws.String(w.uploadID),
			},
		); aErr != nil {
			return errors.Join(err, aErr)
		}
		return err
	}
	return nil
}

// flush flushes all buffers to write a part to S3.
// flush is only called by Write (with both buffers full) and Close/Commit (always).
func (w *writer) flush() error {
	if w.ready.Len() == 0 && w.pending.Len() == 0 {
		return nil
	}

	buf := bytes.NewBuffer(w.ready.data)
	partSize := buf.Len()
	partNumber := aws.Int64(int64(len(w.parts) + 1))

	resp, err := w.driver.S3.UploadPartWithContext(
		w.ctx, &s3.UploadPartInput{
			Bucket:     aws.String(w.driver.Bucket),
			Key:        aws.String(w.key),
			PartNumber: partNumber,
			UploadId:   aws.String(w.uploadID),
			Body:       bytes.NewReader(buf.Bytes()),
		},
	)
	if err != nil {
		return err
	}

	w.parts = append(
		w.parts, &s3.Part{
			ETag:       resp.ETag,
			PartNumber: partNumber,
			Size:       aws.Int64(int64(partSize)),
		},
	)
	// reset the flushed buffer and swap buffers
	w.ready.Clear()
	w.ready, w.pending = w.pending, w.ready

	// In case we have more data in the pending buffer (now ready), we need to flush it
	if w.ready.Len() > 0 {
		err := w.flush()
		return err
	}

	return nil
}
