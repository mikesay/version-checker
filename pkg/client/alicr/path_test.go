package alicr

import "testing"

func TestIsHost(t *testing.T) {
	tests := map[string]struct {
		host  string
		expIs bool
	}{
		"an empty host should be false": {
			host:  "",
			expIs: false,
		},
		"random string should be false": {
			host:  "foobar",
			expIs: false,
		},
		"random string with dots should be false": {
			host:  "foobar.foo",
			expIs: false,
		},
		"registry.aliyuncs.com with no region should be false": {
			host:  "registry.aliyuncs.com",
			expIs: false,
		},
		"registry.region.aliyuncs.com with random region should be true": {
			host:  "registry.cn-shanghai.aliyuncs.com",
			expIs: true,
		},
	}

	handler := new(Client)
	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			if isHost := handler.IsHost(test.host); isHost != test.expIs {
				t.Errorf("%s: unexpected IsHost, exp=%t got=%t",
					test.host, test.expIs, isHost)
			}
		})
	}
}

func TestRepoImage(t *testing.T) {
	tests := map[string]struct {
		path              string
		expRepo, expImage string
	}{
		"single image should return as image": {
			path:     "aliyun-ingress-controller",
			expRepo:  "acs",
			expImage: "aliyun-ingress-controller",
		},
		"two segments to path should return both": {
			path:     "aliacs-app-catalog/istio-operator",
			expRepo:  "aliacs-app-catalog",
			expImage: "istio-operator",
		},
	}

	handler := new(Client)
	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			if repo, image := handler.RepoImageFromPath(test.path); !(repo == test.expRepo &&
				image == test.expImage) {
				t.Errorf("%s: unexpected repo/image, exp=%s,%s got=%s,%s",
					test.path, test.expRepo, test.expImage, repo, image)
			}
		})
	}
}
