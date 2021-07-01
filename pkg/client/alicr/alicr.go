package alicr

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/aliyun/alibaba-cloud-sdk-go/sdk/requests"
	"github.com/aliyun/alibaba-cloud-sdk-go/services/cr"
	"github.com/jetstack/version-checker/pkg/api"
)

type Tag struct {
	Status      string `json:"status"`
	Digest      string `json:"digest"`
	Tag         string `json:"tag"`
	ImageCreate int64  `json:"imageCreate"`
	ImageId     string `json:"imageId"`
	ImageUpdate int64  `json:"imageUpdate"`
	ImageSize   int64  `json:"imageSize"`
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
	cacheMu             sync.Mutex
	cachedRegionClients map[string]*cr.Client

	Options
}

type Options struct {
	AccessKeyID     string
	SecretAccessKey string
}

func New(opts Options) *Client {
	return &Client{
		Options:             opts,
		cachedRegionClients: make(map[string]*cr.Client),
	}
}

func (c *Client) Name() string {
	return "alicr"
}

func (c *Client) Tags(ctx context.Context, host, repo, image string) ([]api.ImageTag, error) {
	var (
		err     error
		client  *cr.Client
		request *cr.GetRepoTagsRequest
		resp    *cr.GetRepoTagsResponse
	)

	matches := alicrPattern.FindStringSubmatch(host)
	if len(matches) < 2 {
		return nil, fmt.Errorf("aliyun client not suitable for image host: %s", host)
	}

	region := matches[1]
	client, err = c.getClient(region)
	if err != nil {
		return nil, fmt.Errorf("failed to construct alicr client for image host %s: %s",
			host, err)
	}

	page := 1
	pageSize := 30
	request = cr.CreateGetRepoTagsRequest()
	request.RepoNamespace = repo
	request.RepoName = image

	var tags []api.ImageTag
	for {
		request.Page = requests.NewInteger(page)
		request.PageSize = requests.NewInteger(pageSize)
		request.ConnectTimeout = time.Minute
		request.ReadTimeout = time.Minute * 2

		resp, err = client.GetRepoTags(request)
		if err != nil {
			return nil, fmt.Errorf("failed to get repo tags of image %s: %s",
				image, err)
		}
		respData := resp.GetHttpContentBytes()
		var repoTagData RepoTagData
		err = json.Unmarshal(respData, &repoTagData)
		if err != nil {
			return nil, fmt.Errorf("failed to construct alicr client for image host %s: %s",
				host, err)
		}

		for _, tg := range repoTagData.Data.Tags {
			tags = append(tags, api.ImageTag{
				SHA:       tg.Digest,
				Timestamp: time.Unix(tg.ImageUpdate, 0),
				Tag:       tg.Tag,
			})
		}

		if repoTagData.Data.Total-page*pageSize <= 0 {
			break
		}

		page = page + 1
	}

	return tags, nil
}

func (c *Client) getClient(region string) (*cr.Client, error) {
	c.cacheMu.Lock()
	defer c.cacheMu.Unlock()

	client, ok := c.cachedRegionClients[region]
	if !ok || client == nil {
		var err error
		client, err = cr.NewClientWithAccessKey(region, c.AccessKeyID, c.SecretAccessKey)
		if err != nil {
			return nil, err
		}
	}

	c.cachedRegionClients[region] = client
	return client, nil
}
