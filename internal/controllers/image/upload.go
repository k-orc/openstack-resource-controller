/*
Copyright 2024 The ORC Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

	http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package image

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/gophercloud/gophercloud/v2/openstack/image/v2/images"
	corev1 "k8s.io/api/core/v1"
	ctrl "sigs.k8s.io/controller-runtime"

	orcv1alpha1 "github.com/k-orc/openstack-resource-controller/v2/api/v1alpha1"
	"github.com/k-orc/openstack-resource-controller/v2/internal/logging"
	osclients "github.com/k-orc/openstack-resource-controller/v2/internal/osclients"
	orcerrors "github.com/k-orc/openstack-resource-controller/v2/internal/util/errors"
)

func (r *orcImageReconciler) hashVerifier(ctx context.Context, orcImage *orcv1alpha1.Image, expectedValue string) hashCompletionHandler {
	log := ctrl.LoggerFrom(ctx)

	return func(hash string) error {
		if hash == expectedValue {
			log.V(logging.Verbose).Info("download hash verification succeeded")
		} else {
			log.V(logging.Info).Info("download hash verification failed", "expected", expectedValue, "got", hash)
			msg := "download hash verification failed. got: " + hash
			r.recorder.Eventf(orcImage, corev1.EventTypeWarning, "HashVerificationFailed", msg)
			return errors.New(msg)
		}
		return nil
	}
}

func (r *orcImageReconciler) downloadProgressReporter(ctx context.Context, orcImage *orcv1alpha1.Image, glanceImage *images.Image, contentLength int64) progressReporter {
	log := ctrl.LoggerFrom(ctx)

	var ofTotal string
	if contentLength > 0 {
		ofTotal = fmt.Sprintf("/%dMB", int(contentLength/1024/1024))
	}

	interval := 10 * time.Second
	nextUpdate := time.Now().Add(interval)
	return func(progress int64) {
		if time.Now().After(nextUpdate) {
			msg := fmt.Sprintf("Downloaded %dMB"+ofTotal, int(progress/1024/1024))
			err := r.updateStatus(ctx, orcImage, withResource(glanceImage),
				withProgressMessage(downloadingMessage(msg, orcImage)))
			if err != nil {
				// Failure to update status here is not fatal
				log.Error(err, "Error writing status during image upload")
			}
			nextUpdate = time.Now().Add(interval)
		}
	}
}

func (r *orcImageReconciler) uploadImageContent(ctx context.Context, orcImage *orcv1alpha1.Image, imageClient osclients.ImageClient, glanceImage *images.Image) (err error) {
	log := ctrl.LoggerFrom(ctx)
	log.V(logging.Info).Info("Uploading image content")

	content, err := requireResourceContent(orcImage)
	if err != nil {
		return err
	}

	download := content.Download
	if download == nil {
		// Should have been caught by validation
		return orcerrors.Terminal(orcv1alpha1.ConditionReasonInvalidConfiguration, "image source type URL has no url entry")
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, download.URL, http.NoBody)
	if err != nil {
		return fmt.Errorf("error creating request for %s: %w", download.URL, err)
	}

	client := http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("error requesting %s: %w", download.URL, err)
	}
	defer func() {
		err = errors.Join(err, resp.Body.Close())
	}()
	log.V(logging.Debug).Info("got response", "status", resp.Status, "contentLength", resp.ContentLength)

	// Report progress while reading downloaded data
	reader := newReaderWithProgress(resp.Body, r.downloadProgressReporter(ctx, orcImage, glanceImage, resp.ContentLength))

	// If the content defines a hash, calculate the hash while downloading and verify it before returning a successful read to glance
	if download.Hash != nil {
		log.V(logging.Verbose).Info("will verify download hash", "algorithm", download.Hash.Algorithm, "value", download.Hash.Value)
		reader, err = newReaderWithHash(reader, download.Hash.Algorithm, r.hashVerifier(ctx, orcImage, download.Hash.Value))
		if err != nil {
			return err
		}
	}

	// Buffer reads
	// This is especially important when using decompression, which can make extremely small read requests
	reader = bufio.NewReaderSize(reader, transferBufferSizeBytes)

	// If the content requires decompression, decompress before sending to glance
	if download.Decompress != nil {
		log.V(logging.Verbose).Info("will decompress downloaded content", "algorithm", *download.Decompress)
		reader, err = newReaderWithDecompression(reader, *download.Decompress)
		if err != nil {
			return fmt.Errorf("opening %s: %w", download.URL, err)
		}
	}

	err = r.updateStatus(ctx, orcImage, withResource(glanceImage),
		withIncrementDownloadAttempts(),
		withProgressMessage(downloadingMessage("Starting image upload", orcImage)))
	if err != nil {
		return err
	}

	err = imageClient.UploadData(ctx, glanceImage.ID, reader)
	if err != nil {
		if orcerrors.IsInvalidError(err) {
			err = orcerrors.Terminal(orcv1alpha1.ConditionReasonInvalidConfiguration, err.Error(), err)
		}
		return fmt.Errorf("error writing data to glance: %w", err)
	}

	return nil
}
