package alicr

import (
	"context"
	"fmt"
	"sync"

	"github.com/aliyun/alibaba-cloud-sdk-go/services/cr"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/ecr"
	"github.com/jetstack/version-checker/pkg/api"
	"github.com/jetstack/version-checker/pkg/client/util"
)

type Tag struct {
	Status      string `json:"status"`
	Digest      string `json:"digest"`
	Tag         string `json:"tag"`
	ImageCreate uint64 `json:"imageCreate"`
	ImageId     string `json:"imageId"`
	ImageUpdate uint64 `json:"imageUpdate"`
	ImageSize   uint64 `json:"imageSize"`
}

type RepoTagRepose struct {
	Page     int   `json:"page"`
	PageSize int   `json:"pageSize"`
	Total    int   `json:"total"`
	Tags     []Tag `json:"tags"`
}

type RepoTagData struct {
	Data RepoTagRepose `json:"data"`
}

type Client struct {
	cacheMu            sync.Mutex
	Options            Options
	cachedRegionClient *cr.Client
}

type Options struct {
	AccessKeyID     string
	SecretAccessKey string
	Region          string
	SessionToken    string
}

func New(opts Options) *Client {
	var (
		client *cr.Client
		err    error
	)
	client, err = cr.NewClientWithAccessKey(opts.Region, opts.AccessKeyID, opts.SecretAccessKey)
	if err != nil {
		// Handle exceptions
		panic(err)
	}

	return &Client{
		Options:            opts,
		cachedRegionClient: client,
	}
}

func (c *Client) Name() string {
	return "alicr"
}

func (c *Client) Tags(ctx context.Context, host, repo, image string) ([]api.ImageTag, error) {
	matches := ecrPattern.FindStringSubmatch(host)
	if len(matches) < 3 {
		return nil, fmt.Errorf("aws client not suitable for image host: %s", host)
	}

	id := matches[1]
	region := matches[3]

	client, err := c.getClient(region)
	if err != nil {
		return nil, fmt.Errorf("failed to construct ecr client for image host %s: %s",
			host, err)
	}

	repoName := util.JoinRepoImage(repo, image)
	images, err := client.DescribeImagesWithContext(ctx, &ecr.DescribeImagesInput{
		RepositoryName: &repoName,
		RegistryId:     aws.String(id),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to describe images: %s", err)
	}

	var tags []api.ImageTag
	for _, img := range images.ImageDetails {

		// Continue early if no tags available
		if len(img.ImageTags) == 0 {
			tags = append(tags, api.ImageTag{
				SHA:       *img.ImageDigest,
				Timestamp: *img.ImagePushedAt,
			})

			continue
		}

		for _, tag := range img.ImageTags {
			tags = append(tags, api.ImageTag{
				SHA:       *img.ImageDigest,
				Timestamp: *img.ImagePushedAt,
				Tag:       *tag,
			})
		}
	}

	return tags, nil
}
