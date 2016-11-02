// Package s3io contains all logic for dealing with s3 files
/*
 * Copyright 2007-2016 Abstrium <contact (at) pydio.com>
 * This file is part of Pydio.
 *
 * Pydio is free software: you can redistribute it and/or modify
 * it under the terms of the GNU Affero General Public License as published by
 * the Free Software Foundation, either version 3 of the License, or
 * (at your option) any later version.
 *
 * Pydio is distributed in the hope that it will be useful,
 * but WITHOUT ANY WARRANTY; without even the implied warranty of
 * MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
 * GNU Affero General Public License for more details.
 *
 * You should have received a copy of the GNU Affero General Public License
 * along with Pydio.  If not, see <http://www.gnu.org/licenses/>.
 *
 * The latest code can be found at <https://pydio.com/>.
 */
package s3io

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/aws/aws-sdk-go/service/s3/s3manager"
	"github.com/pydio/pydio-booster/io"

	pydiolog "github.com/pydio/pydio-booster/log"
)

type s3writer struct {
	totalBytes int64
	l          sync.Mutex
	wwait      sync.Cond

	*io.PipeWriter
}

var log *pydiolog.Logger

func init() {
	log = pydiolog.New(pydiolog.GetLevel(), "[s3io] ", pydiolog.Ldate|pydiolog.Ltime|pydiolog.Lmicroseconds)
}

// Open S3 node as a file
func Open(node *pydio.Node, flag int) (*pydio.File, error) {

	var reader *pydio.Reader
	var writer *s3writer
	var lock chan (int)

	name := filepath.Join(node.Dir.String(), node.Basename)

	// Creating the aws credentials
	creds := credentials.NewStaticCredentials(node.Options.S3Options.APIKey, node.Options.S3Options.SecretKey, "")

	config := aws.NewConfig()
	config = config.WithCredentials(creds)
	config = config.WithRegion(node.Options.S3Options.Region)

	// Creating the aws session
	sess, err := session.NewSession(config)
	if err != nil {
		fmt.Println("failed to create session,", err)
		return nil, err
	}

	// Creating the handlers
	if flag&os.O_RDWR != 0 || flag&os.O_WRONLY == 0 {
		reader = readHandler(sess, name, node.Options.S3Options.Container)
	}

	if flag&os.O_WRONLY != 0 || flag&os.O_RDWR != 0 {
		lock = make(chan (int))
		if flag&os.O_APPEND != 0 {
			writer = appendHandler(sess, name, node.Options.S3Options.Container, lock)
		} else {
			writer = writeHandler(sess, name, node.Options.S3Options.Container, lock)
		}
	}

	return pydio.NewFile(
		node,
		name,
		reader,
		writer,
		lock,
	), nil
}

func readHandler(sess *session.Session, name string, bucket string) *pydio.Reader {

	reader, writer := io.Pipe()

	// Defining the read handler
	go func() {

		defer writer.Close()

		s3Client := s3.New(sess)

		result, err := s3Client.GetObject(&s3.GetObjectInput{
			Bucket: aws.String(bucket),
			Key:    aws.String(name),
		})

		if err != nil {
			log.Errorln(err.Error())
			reader.CloseWithError(errors.New("Could not read from file"))
			return
		}

		io.Copy(writer, result.Body)

	}()

	return &pydio.Reader{PipeReader: reader}
}

func writeHandler(sess *session.Session, name string, bucket string, lock chan (int)) *s3writer {

	reader, writer := io.Pipe()

	// Defining the write handler
	go func() {
		// Releasing the lock
		defer func() {
			reader.Close()
			lock <- 1
		}()

		uploader := s3manager.NewUploader(sess)

		result, err := uploader.Upload(&s3manager.UploadInput{
			Bucket: aws.String(bucket),
			Key:    aws.String(name),
			Body:   reader,
		})

		if err != nil {
			log.Errorln(err.Error())
			writer.CloseWithError(err)
			return
		}

		log.Infoln("Successfully uploaded to ", result.Location)
	}()

	return newS3Writer(writer)
}

func appendHandler(sess *session.Session, name string, bucket string, lock chan (int)) *s3writer {

	reader, writer := io.Pipe()

	var completedParts []*s3.CompletedPart

	go func() {
		defer func() {
			reader.Close()
			lock <- 1
		}()

		var err error

		s3Client := s3.New(sess)

		// Create Multipart Request
		var createOutput *s3.CreateMultipartUploadOutput

		createInput := &s3.CreateMultipartUploadInput{
			Bucket: aws.String(bucket),
			Key:    aws.String(name),
		}

		if createOutput, err = s3Client.CreateMultipartUpload(createInput); err != nil {
			log.Errorln(err.Error())
			writer.CloseWithError(err)
			return
		}

		// Upload Part 1
		var uploadPart1CopyOutput *s3.UploadPartCopyOutput

		uploadPart1CopyInput := &s3.UploadPartCopyInput{
			Bucket:     aws.String(bucket),
			CopySource: aws.String("/" + bucket + "/" + url.QueryEscape(strings.TrimLeft(name, "/"))),
			Key:        aws.String(name),
			PartNumber: aws.Int64(1),
			UploadId:   createOutput.UploadId,
		}

		if uploadPart1CopyOutput, err = s3Client.UploadPartCopy(uploadPart1CopyInput); err != nil {
			log.Errorln(err.Error())
			writer.CloseWithError(err)
			return
		}

		completed := &s3.CompletedPart{ETag: uploadPart1CopyOutput.CopyPartResult.ETag, PartNumber: aws.Int64(1)}
		completedParts = append(completedParts, completed)

		// Upload Part2
		data, _ := ioutil.ReadAll(reader)
		byteReader := bytes.NewReader(data)

		var uploadPart2Output *s3.UploadPartOutput

		uploadPart2Input := &s3.UploadPartInput{
			Bucket:     aws.String(bucket),
			Key:        aws.String(name),
			PartNumber: aws.Int64(2),
			UploadId:   createOutput.UploadId,
			Body:       byteReader,
		}

		if uploadPart2Output, err = s3Client.UploadPart(uploadPart2Input); err != nil {
			log.Errorln(err.Error())
			writer.CloseWithError(err)
			return
		}

		completed = &s3.CompletedPart{ETag: uploadPart2Output.ETag, PartNumber: aws.Int64(2)}
		completedParts = append(completedParts, completed)

		// Completing Part
		var completeUploadOutput *s3.CompleteMultipartUploadOutput

		completeUploadInput := &s3.CompleteMultipartUploadInput{
			Bucket:          aws.String(bucket),
			Key:             aws.String(name),
			UploadId:        createOutput.UploadId,
			MultipartUpload: &s3.CompletedMultipartUpload{Parts: completedParts},
		}

		if completeUploadOutput, err = s3Client.CompleteMultipartUpload(completeUploadInput); err != nil {
			log.Errorln(err.Error())
			writer.CloseWithError(err)
			return
		}

		log.Infoln("Successul upload ", completeUploadOutput)
	}()

	return newS3Writer(writer)
}

func newS3Writer(w *io.PipeWriter) (writer *s3writer) {

	writer = &s3writer{PipeWriter: w}

	writer.wwait.L = &writer.l

	return
}
func (writer *s3writer) Close() error {
	if writer != nil {
		return writer.PipeWriter.Close()
	}

	return nil
}

func (writer *s3writer) WriteAt(p []byte, off int64) (n int, err error) {
	writer.l.Lock()
	for {
		// Waiting for our turn (the pipewriter to have enough bytes in pipe)
		if writer.totalBytes >= off {
			break
		}

		writer.wwait.Wait()
	}

	n, err = writer.PipeWriter.Write(p)

	writer.totalBytes += int64(n)
	writer.l.Unlock()
	writer.wwait.Broadcast()

	return
}
