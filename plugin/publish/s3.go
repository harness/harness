package publish

import (
	"fmt"
	"strings"

	"github.com/drone/drone/plugin/condition"
	"github.com/drone/drone/shared/build/buildfile"
)

type S3 struct {
	Key    string `yaml:"access_key,omitempty"`
	Secret string `yaml:"secret_key,omitempty"`
	Bucket string `yaml:"bucket,omitempty"`

	// us-east-1
	// us-west-1
	// us-west-2
	// eu-west-1
	// ap-southeast-1
	// ap-southeast-2
	// ap-northeast-1
	// sa-east-1
	Region string `yaml:"region,omitempty"`

	// Indicates the files ACL, which should be one
	// of the following:
	//     private
	//     public-read
	//     public-read-write
	//     authenticated-read
	//     bucket-owner-read
	//     bucket-owner-full-control
	Access string `yaml:"acl,omitempty"`

	// Copies the files from the specified directory.
	// Regexp matching will apply to match multiple
	// files
	//
	// Examples:
	//    /path/to/file
	//    /path/to/*.txt
	//    /path/to/*/*.txt
	//    /path/to/**
	Source string `yaml:"source,omitempty"`
	Target string `yaml:"target,omitempty"`

	// Recursive uploads
	Recursive bool `yaml:"recursive"`

	Condition *condition.Condition `yaml:"when,omitempty"`
}

func (s *S3) Write(f *buildfile.Buildfile) {

	// skip if AWS key or SECRET are empty. A good example for this would
	// be forks building a project. S3 might be configured in the source
	// repo, but not in the fork
	if len(s.Key) == 0 || len(s.Secret) == 0 {
		return
	}

	// debugging purposes so we can see if / where something is failing
	f.WriteCmdSilent("echo 'publishing to Amazon S3 ...'")

	// install the AWS cli using PIP
	f.WriteCmdSilent("[ -f /usr/bin/sudo ] || pip install awscli 1> /dev/null 2> /dev/null")
	f.WriteCmdSilent("[ -f /usr/bin/sudo ] && sudo pip install awscli 1> /dev/null 2> /dev/null")

	f.WriteEnv("AWS_ACCESS_KEY_ID", s.Key)
	f.WriteEnv("AWS_SECRET_ACCESS_KEY", s.Secret)

	// make sure a default region is set
	if len(s.Region) == 0 {
		s.Region = "us-east-1"
	}

	// make sure a default access is set
	// let's be conservative and assume private
	if len(s.Access) == 0 {
		s.Access = "private"
	}

	// if the target starts with a "/" we need
	// to remove it, otherwise we might adding
	// a 3rd slash to s3://
	if strings.HasPrefix(s.Target, "/") {
		s.Target = s.Target[1:]
	}

	switch s.Recursive {
	case true:
		f.WriteCmd(fmt.Sprintf(`aws s3 cp %s s3://%s/%s --recursive --acl %s --region %s`, s.Source, s.Bucket, s.Target, s.Access, s.Region))
	case false:
		f.WriteCmd(fmt.Sprintf(`aws s3 cp %s s3://%s/%s --acl %s --region %s`, s.Source, s.Bucket, s.Target, s.Access, s.Region))
	}
}

func (s *S3) GetCondition() *condition.Condition {
	return s.Condition
}
