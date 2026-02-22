//  Copyright 2023 Harness, Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package config

import "github.com/harness/gitness/types"

func GetS3StorageParameters(c *types.Config) map[string]any {
	s3Properties := make(map[string]any)
	s3Properties["accesskey"] = c.Registry.Storage.S3Storage.AccessKey
	s3Properties["secretkey"] = c.Registry.Storage.S3Storage.SecretKey
	s3Properties["region"] = c.Registry.Storage.S3Storage.Region
	s3Properties["regionendpoint"] = c.Registry.Storage.S3Storage.RegionEndpoint
	s3Properties["forcepathstyle"] = c.Registry.Storage.S3Storage.ForcePathStyle
	s3Properties["accelerate"] = c.Registry.Storage.S3Storage.Accelerate
	s3Properties["bucket"] = c.Registry.Storage.S3Storage.Bucket
	s3Properties["encrypt"] = c.Registry.Storage.S3Storage.Encrypt
	s3Properties["keyid"] = c.Registry.Storage.S3Storage.KeyID
	s3Properties["secure"] = c.Registry.Storage.S3Storage.Secure
	s3Properties["v4auth"] = c.Registry.Storage.S3Storage.V4Auth
	s3Properties["chunksize"] = c.Registry.Storage.S3Storage.ChunkSize
	s3Properties["multipartcopychunksize"] = c.Registry.Storage.S3Storage.MultipartCopyChunkSize
	s3Properties["multipartcopymaxconcurrency"] = c.Registry.Storage.S3Storage.MultipartCopyMaxConcurrency
	s3Properties["multipartcopythresholdsize"] = c.Registry.Storage.S3Storage.MultipartCopyThresholdSize
	s3Properties["rootdirectory"] = c.Registry.Storage.S3Storage.RootDirectory
	s3Properties["usedualstack"] = c.Registry.Storage.S3Storage.UseDualStack
	s3Properties["loglevel"] = c.Registry.Storage.S3Storage.LogLevel
	return s3Properties
}

func GetFilesystemParams(c *types.Config) map[string]any {
	props := make(map[string]any)
	props["maxthreads"] = c.Registry.Storage.FileSystemStorage.MaxThreads
	props["rootdirectory"] = c.Registry.Storage.FileSystemStorage.RootDirectory
	return props
}

func GetGCSStorageParameters(c *types.Config) map[string]any {
	gcsProperties := make(map[string]any)
	gcsProperties["bucket"] = c.Registry.Storage.GCSStorage.Bucket
	return gcsProperties
}
