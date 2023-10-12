/*
Copyright 2014 The Kubernetes Authors.

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

package filters

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"errors"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"golang.org/x/net/http2"

	"k8s.io/apiserver/pkg/authentication/authenticator"
	"k8s.io/apiserver/pkg/authentication/request/anonymous"
	"k8s.io/apiserver/pkg/authentication/user"
	genericapirequest "k8s.io/apiserver/pkg/endpoints/request"
	"k8s.io/apiserver/pkg/features"
	utilfeature "k8s.io/apiserver/pkg/util/feature"
	"k8s.io/client-go/kubernetes/scheme"
	featuregatetesting "k8s.io/component-base/featuregate/testing"
)

func TestAuthenticateRequestWithAud(t *testing.T) {
	success, failed := 0, 0
	testcases := []struct {
		name          string
		apiAuds       []string
		respAuds      []string
		expectSuccess bool
	}{
		{
			name:          "no api audience and no audience in response",
			apiAuds:       nil,
			respAuds:      nil,
			expectSuccess: true,
		},
		{
			name:          "audience in response",
			apiAuds:       nil,
			respAuds:      []string{"other"},
			expectSuccess: true,
		},
		{
			name:          "with api audience",
			apiAuds:       authenticator.Audiences([]string{"other"}),
			respAuds:      nil,
			expectSuccess: true,
		},
		{
			name:          "api audience matching response audience",
			apiAuds:       authenticator.Audiences([]string{"other"}),
			respAuds:      []string{"other"},
			expectSuccess: true,
		},
		{
			name:          "api audience non-matching response audience",
			apiAuds:       authenticator.Audiences([]string{"other"}),
			respAuds:      []string{"some"},
			expectSuccess: false,
		},
	}
	for _, tc := range testcases {
		t.Run(tc.name, func(t *testing.T) {
			success, failed = 0, 0
			auth := WithAuthentication(
				http.HandlerFunc(func(_ http.ResponseWriter, req *http.Request) {
					if tc.expectSuccess {
						success = 1
					} else {
						t.Errorf("unexpected call to handler")
					}
				}),
				authenticator.RequestFunc(func(req *http.Request) (*authenticator.Response, bool, error) {
					if req.Header.Get("Authorization") == "Something" {
						return &authenticator.Response{User: &user.DefaultInfo{Name: "user"}, Audiences: authenticator.Audiences(tc.respAuds)}, true, nil
					}
					return nil, false, errors.New("Authorization header is missing.")
				}),
				http.HandlerFunc(func(_ http.ResponseWriter, _ *http.Request) {
					if tc.expectSuccess {
						t.Errorf("unexpected call to failed")
					} else {
						failed = 1
					}
				}),
				tc.apiAuds,
			)
			auth.ServeHTTP(httptest.NewRecorder(), &http.Request{Header: map[string][]string{"Authorization": {"Something"}}})
			if tc.expectSuccess {
				assert.Equal(t, 1, success)
				assert.Equal(t, 0, failed)
			} else {
				assert.Equal(t, 0, success)
				assert.Equal(t, 1, failed)
			}
		})
	}
}

func TestAuthenticateMetrics(t *testing.T) {
	testcases := []struct {
		name         string
		header       bool
		apiAuds      []string
		respAuds     []string
		expectMetric bool
		expectOk     bool
		expectError  bool
	}{
		{
			name:        "no api audience and no audience in response",
			header:      true,
			apiAuds:     nil,
			respAuds:    nil,
			expectOk:    true,
			expectError: false,
		},
		{
			name:        "api audience matching response audience",
			header:      true,
			apiAuds:     authenticator.Audiences([]string{"other"}),
			respAuds:    []string{"other"},
			expectOk:    true,
			expectError: false,
		},
		{
			name:        "no intersection results in error",
			header:      true,
			apiAuds:     authenticator.Audiences([]string{"other"}),
			respAuds:    []string{"some"},
			expectOk:    true,
			expectError: true,
		},
		{
			name:        "no header results in error",
			header:      false,
			apiAuds:     authenticator.Audiences([]string{"other"}),
			respAuds:    []string{"some"},
			expectOk:    false,
			expectError: true,
		},
	}
	for _, tc := range testcases {
		t.Run(tc.name, func(t *testing.T) {
			called := 0
			auth := withAuthentication(
				http.HandlerFunc(func(_ http.ResponseWriter, req *http.Request) {
				}),
				authenticator.RequestFunc(func(req *http.Request) (*authenticator.Response, bool, error) {
					if req.Header.Get("Authorization") == "Something" {
						return &authenticator.Response{User: &user.DefaultInfo{Name: "user"}, Audiences: authenticator.Audiences(tc.respAuds)}, true, nil
					}
					return nil, false, errors.New("Authorization header is missing.")
				}),
				http.HandlerFunc(func(_ http.ResponseWriter, _ *http.Request) {
				}),
				tc.apiAuds,
				func(ctx context.Context, resp *authenticator.Response, ok bool, err error, apiAudiences authenticator.Audiences, authStart time.Time, authFinish time.Time) {
					called = 1
					if tc.expectOk != ok {
						t.Errorf("unexpected value of ok argument: %t", ok)
					}
					if tc.expectError {
						if err == nil {
							t.Errorf("unexpected value of err argument: %s", err)
						}
					} else {
						if err != nil {
							t.Errorf("unexpected value of err argument: %s", err)
						}
					}
				},
			)
			if tc.header {
				auth.ServeHTTP(httptest.NewRecorder(), &http.Request{Header: map[string][]string{"Authorization": {"Something"}}})
			} else {
				auth.ServeHTTP(httptest.NewRecorder(), &http.Request{})
			}
			assert.Equal(t, 1, called)
		})
	}
}

func TestAuthenticateRequest(t *testing.T) {
	success := make(chan struct{})
	auth := WithAuthentication(
		http.HandlerFunc(func(_ http.ResponseWriter, req *http.Request) {
			ctx := req.Context()
			user, ok := genericapirequest.UserFrom(ctx)
			if user == nil || !ok {
				t.Errorf("no user stored in context: %#v", ctx)
			}
			if req.Header.Get("Authorization") != "" {
				t.Errorf("Authorization header should be removed from request on success: %#v", req)
			}
			close(success)
		}),
		authenticator.RequestFunc(func(req *http.Request) (*authenticator.Response, bool, error) {
			if req.Header.Get("Authorization") == "Something" {
				return &authenticator.Response{User: &user.DefaultInfo{Name: "user"}}, true, nil
			}
			return nil, false, errors.New("Authorization header is missing.")
		}),
		http.HandlerFunc(func(_ http.ResponseWriter, _ *http.Request) {
			t.Errorf("unexpected call to failed")
		}),
		nil,
	)

	auth.ServeHTTP(httptest.NewRecorder(), &http.Request{Header: map[string][]string{"Authorization": {"Something"}}})

	<-success
}

func TestAuthenticateRequestFailed(t *testing.T) {
	failed := make(chan struct{})
	auth := WithAuthentication(
		http.HandlerFunc(func(_ http.ResponseWriter, req *http.Request) {
			t.Errorf("unexpected call to handler")
		}),
		authenticator.RequestFunc(func(req *http.Request) (*authenticator.Response, bool, error) {
			return nil, false, nil
		}),
		http.HandlerFunc(func(_ http.ResponseWriter, _ *http.Request) {
			close(failed)
		}),
		nil,
	)

	auth.ServeHTTP(httptest.NewRecorder(), &http.Request{})

	<-failed
}

func TestAuthenticateRequestError(t *testing.T) {
	failed := make(chan struct{})
	auth := WithAuthentication(
		http.HandlerFunc(func(_ http.ResponseWriter, req *http.Request) {
			t.Errorf("unexpected call to handler")
		}),
		authenticator.RequestFunc(func(req *http.Request) (*authenticator.Response, bool, error) {
			return nil, false, errors.New("failure")
		}),
		http.HandlerFunc(func(_ http.ResponseWriter, _ *http.Request) {
			close(failed)
		}),
		nil,
	)

	auth.ServeHTTP(httptest.NewRecorder(), &http.Request{})

	<-failed
}

func TestUnauthenticatedHTTP2ClientConnectionClose(t *testing.T) {
	s := httptest.NewUnstartedServer(WithAuthentication(
		http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { _, _ = w.Write([]byte("ok")) }),
		authenticator.RequestFunc(func(r *http.Request) (*authenticator.Response, bool, error) {
			switch r.Header.Get("Authorization") {
			case "known":
				return &authenticator.Response{User: &user.DefaultInfo{Name: "panda"}}, true, nil
			case "error":
				return nil, false, errors.New("authn err")
			case "anonymous":
				return anonymous.NewAuthenticator().AuthenticateRequest(r)
			case "anonymous_group":
				return &authenticator.Response{User: &user.DefaultInfo{Groups: []string{user.AllUnauthenticated}}}, true, nil
			default:
				return nil, false, nil
			}
		}),
		http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			r = r.WithContext(genericapirequest.WithRequestInfo(r.Context(), &genericapirequest.RequestInfo{}))
			Unauthorized(scheme.Codecs).ServeHTTP(w, r)
		}),
		nil,
	))

	http2Options := &http2.Server{}

	if err := http2.ConfigureServer(s.Config, http2Options); err != nil {
		t.Fatal(err)
	}

	s.TLS = s.Config.TLSConfig

	s.StartTLS()
	t.Cleanup(s.Close)

	const reqs = 4

	cases := []struct {
		name                   string
		authorizationHeader    string
		skipHTTP2DOSMitigation bool
		expectConnections      uint64
	}{
		{
			name:                   "known",
			authorizationHeader:    "known",
			skipHTTP2DOSMitigation: false,
			expectConnections:      1,
		},
		{
			name:                   "error",
			authorizationHeader:    "error",
			skipHTTP2DOSMitigation: false,
			expectConnections:      reqs,
		},
		{
			name:                   "anonymous",
			authorizationHeader:    "anonymous",
			skipHTTP2DOSMitigation: false,
			expectConnections:      reqs,
		},
		{
			name:                   "anonymous_group",
			authorizationHeader:    "anonymous_group",
			skipHTTP2DOSMitigation: false,
			expectConnections:      reqs,
		},
		{
			name:                   "other",
			authorizationHeader:    "other",
			skipHTTP2DOSMitigation: false,
			expectConnections:      reqs,
		},

		{
			name:                   "known skip=true",
			authorizationHeader:    "known",
			skipHTTP2DOSMitigation: true,
			expectConnections:      1,
		},
		{
			name:                   "error skip=true",
			authorizationHeader:    "error",
			skipHTTP2DOSMitigation: true,
			expectConnections:      1,
		},
		{
			name:                   "anonymous skip=true",
			authorizationHeader:    "anonymous",
			skipHTTP2DOSMitigation: true,
			expectConnections:      1,
		},
		{
			name:                   "anonymous_group skip=true",
			authorizationHeader:    "anonymous_group",
			skipHTTP2DOSMitigation: true,
			expectConnections:      1,
		},
		{
			name:                   "other skip=true",
			authorizationHeader:    "other",
			skipHTTP2DOSMitigation: true,
			expectConnections:      1,
		},
	}

	rootCAs := x509.NewCertPool()
	rootCAs.AddCert(s.Certificate())

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			f := func(t *testing.T, nextProto string, expectConnections uint64) {
				defer featuregatetesting.SetFeatureGateDuringTest(t, utilfeature.DefaultFeatureGate, features.UnauthenticatedHTTP2DOSMitigation, !tc.skipHTTP2DOSMitigation)()

				var localAddrs atomic.Uint64 // indicates how many TCP connection set up

				tlsConfig := &tls.Config{
					RootCAs:    rootCAs,
					NextProtos: []string{nextProto},
				}

				dailer := tls.Dialer{
					Config: tlsConfig,
				}

				tr := &http.Transport{
					TLSHandshakeTimeout: 10 * time.Second,
					TLSClientConfig:     tlsConfig,
					DialTLSContext: func(ctx context.Context, network, addr string) (net.Conn, error) {
						conn, err := dailer.DialContext(ctx, network, addr)
						if err != nil {
							return nil, err
						}

						localAddrs.Add(1)

						return conn, nil
					},
				}

				tr.MaxIdleConnsPerHost = 1 // allow http1 to have keep alive connections open
				if nextProto == http2.NextProtoTLS {
					// Disable connection pooling to avoid additional connections
					// that cause the test to flake
					tr.MaxIdleConnsPerHost = -1
					if err := http2.ConfigureTransport(tr); err != nil {
						t.Fatal(err)
					}
				}

				client := &http.Client{
					Transport: tr,
				}

				for i := 0; i < reqs; i++ {
					req, err := http.NewRequest(http.MethodGet, s.URL, nil)
					if err != nil {
						t.Fatal(err)
					}
					if len(tc.authorizationHeader) > 0 {
						req.Header.Set("Authorization", tc.authorizationHeader)
					}

					resp, err := client.Do(req)
					if err != nil {
						t.Fatal(err)
					}
					_, _ = io.Copy(io.Discard, resp.Body)
					_ = resp.Body.Close()
				}

				if expectConnections != localAddrs.Load() {
					t.Fatalf("expect TCP connection: %d, actual: %d", expectConnections, localAddrs.Load())
				}
			}

			t.Run(http2.NextProtoTLS, func(t *testing.T) {
				f(t, http2.NextProtoTLS, tc.expectConnections)
			})

			// http1 connection reuse occasionally flakes on CI, skipping for now
			// t.Run("http/1.1", func(t *testing.T) {
			// 	f(t, "http/1.1", 1)
			// })
		})
	}
}
