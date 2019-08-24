package pdfgenerator

import (
	"os"
	"path/filepath"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3/s3manager"
)

func S3Upload(region, bucket, path string) error {
	di, err := newDirectoryIterator(bucket, path)
	if err != nil {
		return err
	}

	sess, err := session.NewSession(&aws.Config{
		Region: aws.String(region)},
	)
	if err != nil {
		return err
	}

	uploader := s3manager.NewUploader(sess)

	if err := uploader.UploadWithIterator(aws.BackgroundContext(), di); err != nil {
		return err
	}
	return nil
}

type DirectoryIterator struct {
	filePaths []string
	bucket    string
	next      struct {
		path string
		f    *os.File
	}
	err error
}

func newDirectoryIterator(bucket, dir string) (s3manager.BatchUploadIterator, error) {
	var paths []string

	if err := filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if !info.IsDir() {
			paths = append(paths, path)
		}
		return nil
	}); nil != err {
		return nil, err
	}

	return &DirectoryIterator{
		filePaths: paths,
		bucket:    bucket,
	}, nil
}

// Next returns whether next file exists or not
func (di *DirectoryIterator) Next() bool {
	if len(di.filePaths) == 0 {
		di.next.f = nil
		return false
	}

	f, err := os.Open(di.filePaths[0])
	di.err = err
	di.next.f = f
	di.next.path = di.filePaths[0]
	di.filePaths = di.filePaths[1:]

	return di.Err() == nil
}

// Err returns error of DirectoryIterator
func (di *DirectoryIterator) Err() error {
	return di.err
}

// UploadObject uploads a file
func (di *DirectoryIterator) UploadObject() s3manager.BatchUploadObject {
	f := di.next.f
	path := "/" + filepath.Base(filepath.Dir(di.next.path)) + "/" + filepath.Base(di.next.path)
	return s3manager.BatchUploadObject{
		Object: &s3manager.UploadInput{
			Bucket: &di.bucket,
			Key:    &path,
			Body:   f,
		},
		After: func() error {
			return f.Close()
		},
	}
}
