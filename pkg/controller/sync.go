package controller

import (
	"context"
	"fmt"
	"strings"

	"github.com/sirupsen/logrus"
	corev1 "k8s.io/api/core/v1"

	"github.com/mikesay/version-checker/pkg/api"
	"github.com/mikesay/version-checker/pkg/controller/options"
	versionerrors "github.com/mikesay/version-checker/pkg/version/errors"
)

// sync will enqueue a given pod to run against the version checker.
func (c *Controller) sync(ctx context.Context, pod *corev1.Pod) error {
	log := c.log.WithField("name", pod.Name).WithField("namespace", pod.Namespace)

	builder := options.New(pod.Annotations)

	var errs []string
	for _, container := range pod.Spec.Containers {
		if err := c.syncContainer(ctx, log, builder, pod, &container); err != nil {
			errs = append(errs, err.Error())
		}
	}

	if len(errs) > 0 {
		return fmt.Errorf("failed to sync pod %s/%s: %s",
			pod.Name, pod.Namespace, strings.Join(errs, ","))
	}

	return nil
}

// syncContainer will enqueue a given container to check the version
func (c *Controller) syncContainer(ctx context.Context, log *logrus.Entry, builder *options.Builder, pod *corev1.Pod,
	container *corev1.Container) error {
	// If not enabled, exit early
	if !builder.IsEnabled(c.defaultTestAll, container.Name) {
		c.metrics.RemoveImage(pod.Namespace, pod.Name, container.Name)
		return nil
	}

	opts, err := builder.Options(container.Name)
	if err != nil {
		return fmt.Errorf("failed to build options from annotations for %q: %s",
			container.Name, err)
	}

	log = log.WithField("container", container.Name)
	log.Debug("processing conainer image")

	err = c.checkContainer(ctx, log, pod, container, opts)
	// Don't re-sync, if no version found meeting search criteria
	if versionerrors.IsNoVersionFound(err) {
		log.Error(err.Error())
		return nil
	}
	if err != nil {
		return fmt.Errorf("failed to check container image %q: %s",
			container.Name, err)
	}

	return nil
}

// checkContainer will check the given container and options, and update
// metrics according to the result.
func (c *Controller) checkContainer(ctx context.Context, log *logrus.Entry, pod *corev1.Pod,
	container *corev1.Container, opts *api.Options) error {
	result, err := c.checker.Container(ctx, log, pod, container, opts)
	if err != nil {
		return err
	}

	// If no result ready yet, exit early
	if result == nil {
		return nil
	}

	if result.IsLatest {
		log.Debugf("image is latest %s:%s",
			result.ImageURL, result.CurrentVersion)
	} else {
		log.Debugf("image is not latest %s: %s -> %s",
			result.ImageURL, result.CurrentVersion, result.LatestVersion)
	}

	c.metrics.AddImage(pod.Namespace, pod.Name,
		container.Name, result.ImageURL, result.IsLatest,
		result.CurrentVersion, result.LatestVersion)

	return nil
}
