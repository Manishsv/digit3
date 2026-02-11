package minio

import (
	"context"
	"io"

	"github.com/minio/minio-go/v7"
)

func GetFileStreamWithSize(bucket, object string) (io.ReadCloser, int64, error) {
	obj, err := Client.GetObject(context.Background(), bucket, object, minio.GetObjectOptions{})
	if err != nil {
		return nil, 0, err
	}

	info, err := obj.Stat()
	if err != nil {
		obj.Close()
		return nil, 0, err
	}

	return obj, info.Size, nil
}

func CopyObject(srcBucket, destBucket, key string) error {
	src := minio.CopySrcOptions{Bucket: srcBucket, Object: key}
	dest := minio.CopyDestOptions{Bucket: destBucket, Object: key}
	_, err := Client.CopyObject(context.Background(), dest, src)
	return err
}

func DeleteObject(bucket, key string) error {
	return Client.RemoveObject(context.Background(), bucket, key, minio.RemoveObjectOptions{})
}
