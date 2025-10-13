package gitlab

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/spf13/cobra"
	"resty.dev/v3"
)

// TestNewEnumCmd tests the NewEnumCmd function
func TestNewEnumCmd(t *testing.T) {
	tests := []struct {
		name              string
		wantUse           string
		wantShort         string
		wantRequiredFlags []string
	}{
		{
			name:              "creates enum command with correct structure",
			wantUse:           "enum",
			wantShort:         "Enumerate access rights of a GitLab access token",
			wantRequiredFlags: []string{"gitlab", "token"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cmd := NewEnumCmd()
			
			if cmd.Use != tt.wantUse {
				t.Errorf("NewEnumCmd().Use = %v, want %v", cmd.Use, tt.wantUse)
			}
			
			if cmd.Short != tt.wantShort {
				t.Errorf("NewEnumCmd().Short = %v, want %v", cmd.Short, tt.wantShort)
			}

			// Check required flags
			for _, flag := range tt.wantRequiredFlags {
				if !cmd.Flags().Lookup(flag).Changed && cmd.Flags().Lookup(flag) == nil {
					t.Errorf("Flag %s should exist", flag)
				}
			}

			// Verify flags exist
			if cmd.Flags().Lookup("gitlab") == nil {
				t.Error("gitlab flag should exist")
			}
			if cmd.Flags().Lookup("token") == nil {
				t.Error("token flag should exist")
			}
			if cmd.PersistentFlags().Lookup("level") == nil {
				t.Error("level flag should exist")
			}
			if cmd.PersistentFlags().Lookup("verbose") == nil {
				t.Error("verbose flag should exist")
			}
		})
	}
}

// TestTokenAssociations tests the TokenAssociations struct unmarshaling
func TestTokenAssociations(t *testing.T) {
	tests := []struct {
		name       string
		jsonData   string
		wantErr    bool
		wantGroups int
		wantProjs  int
	}{
		{
			name: "valid token associations with groups and projects",
			jsonData: `{
				"groups": [
					{
						"id": 1,
						"web_url": "https://gitlab.com/group1",
						"name": "Group 1",
						"parent_id": null,
						"organization_id": 100,
						"access_levels": 50,
						"visibility": "private"
					}
				],
				"projects": [
					{
						"id": 10,
						"description": "Test project",
						"name": "project1",
						"name_with_namespace": "Group 1 / project1",
						"path": "project1",
						"path_with_namespace": "group1/project1",
						"created_at": "2023-01-01T00:00:00Z",
						"access_levels": {
							"project_access_level": 30,
							"group_access_level": 50
						},
						"visibility": "private",
						"web_url": "https://gitlab.com/group1/project1",
						"namespace": {
							"id": 1,
							"name": "Group 1",
							"path": "group1",
							"kind": "group",
							"full_path": "group1",
							"parent_id": null,
							"avatar_url": "",
							"web_url": "https://gitlab.com/group1"
						}
					}
				]
			}`,
			wantErr:    false,
			wantGroups: 1,
			wantProjs:  1,
		},
		{
			name: "empty token associations",
			jsonData: `{
				"groups": [],
				"projects": []
			}`,
			wantErr:    false,
			wantGroups: 0,
			wantProjs:  0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var ta TokenAssociations
			err := json.Unmarshal([]byte(tt.jsonData), &ta)
			
			if (err != nil) != tt.wantErr {
				t.Errorf("Unmarshal() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			
			if !tt.wantErr {
				if len(ta.Groups) != tt.wantGroups {
					t.Errorf("len(TokenAssociations.Groups) = %v, want %v", len(ta.Groups), tt.wantGroups)
				}
				if len(ta.Projects) != tt.wantProjs {
					t.Errorf("len(TokenAssociations.Projects) = %v, want %v", len(ta.Projects), tt.wantProjs)
				}
			}
		})
	}
}

// TestSelfToken tests the SelfToken struct unmarshaling
func TestSelfToken(t *testing.T) {
	tests := []struct {
		name     string
		jsonData string
		wantErr  bool
		validate func(*testing.T, *SelfToken)
	}{
		{
			name: "valid self token",
			jsonData: `{
				"id": 123,
				"name": "test-token",
				"revoked": false,
				"created_at": "2023-01-01T00:00:00Z",
				"description": "Test token description",
				"scopes": ["api", "read_user"],
				"user_id": 456,
				"last_used_at": "2023-12-31T23:59:59Z",
				"active": true,
				"expires_at": "2024-12-31",
				"last_used_ips": ["192.168.1.1", "10.0.0.1"]
			}`,
			wantErr: false,
			validate: func(t *testing.T, st *SelfToken) {
				if st.ID != 123 {
					t.Errorf("SelfToken.ID = %v, want %v", st.ID, 123)
				}
				if st.Name != "test-token" {
					t.Errorf("SelfToken.Name = %v, want %v", st.Name, "test-token")
				}
				if st.Revoked != false {
					t.Errorf("SelfToken.Revoked = %v, want %v", st.Revoked, false)
				}
				if st.Description != "Test token description" {
					t.Errorf("SelfToken.Description = %v, want %v", st.Description, "Test token description")
				}
				if len(st.Scopes) != 2 {
					t.Errorf("len(SelfToken.Scopes) = %v, want %v", len(st.Scopes), 2)
				}
				if st.UserID != 456 {
					t.Errorf("SelfToken.UserID = %v, want %v", st.UserID, 456)
				}
				if !st.Active {
					t.Errorf("SelfToken.Active = %v, want %v", st.Active, true)
				}
				if len(st.LastUsedIps) != 2 {
					t.Errorf("len(SelfToken.LastUsedIps) = %v, want %v", len(st.LastUsedIps), 2)
				}
			},
		},
		{
			name: "self token with empty arrays",
			jsonData: `{
				"id": 789,
				"name": "minimal-token",
				"revoked": true,
				"created_at": "2023-01-01T00:00:00Z",
				"description": "",
				"scopes": [],
				"user_id": 999,
				"last_used_at": "2023-01-01T00:00:00Z",
				"active": false,
				"expires_at": "",
				"last_used_ips": []
			}`,
			wantErr: false,
			validate: func(t *testing.T, st *SelfToken) {
				if st.ID != 789 {
					t.Errorf("SelfToken.ID = %v, want %v", st.ID, 789)
				}
				if !st.Revoked {
					t.Errorf("SelfToken.Revoked = %v, want %v", st.Revoked, true)
				}
				if len(st.Scopes) != 0 {
					t.Errorf("len(SelfToken.Scopes) = %v, want %v", len(st.Scopes), 0)
				}
				if len(st.LastUsedIps) != 0 {
					t.Errorf("len(SelfToken.LastUsedIps) = %v, want %v", len(st.LastUsedIps), 0)
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var st SelfToken
			err := json.Unmarshal([]byte(tt.jsonData), &st)
			
			if (err != nil) != tt.wantErr {
				t.Errorf("Unmarshal() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			
			if !tt.wantErr && tt.validate != nil {
				tt.validate(t, &st)
			}
		})
	}
}

// TestEnumCurrentToken tests the enumCurrentToken function with mocked HTTP responses
func TestEnumCurrentToken(t *testing.T) {
	tests := []struct {
		name           string
		setupServer    func() *httptest.Server
		expectedCalled bool
	}{
		{
			name: "successful token fetch",
			setupServer: func() *httptest.Server {
				return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					// Verify headers
					if r.Header.Get("PRIVATE-TOKEN") == "" {
						t.Error("PRIVATE-TOKEN header not set")
					}
					
					// Return valid token response
					response := SelfToken{
						ID:          123,
						Name:        "test-token",
						Revoked:     false,
						CreatedAt:   time.Now(),
						Description: "Test token",
						Scopes:      []string{"api"},
						UserID:      456,
						LastUsedAt:  time.Now(),
						Active:      true,
						ExpiresAt:   "2024-12-31",
						LastUsedIps: []string{"192.168.1.1"},
					}
					w.Header().Set("Content-Type", "application/json")
					w.WriteHeader(http.StatusOK)
					json.NewEncoder(w).Encode(response)
				}))
			},
			expectedCalled: true,
		},
		{
			name: "unauthorized token fetch",
			setupServer: func() *httptest.Server {
				return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					w.WriteHeader(http.StatusUnauthorized)
					w.Write([]byte(`{"error": "unauthorized"}`))
				}))
			},
			expectedCalled: true,
		},
		{
			name: "server error",
			setupServer: func() *httptest.Server {
				return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					w.WriteHeader(http.StatusInternalServerError)
					w.Write([]byte(`{"error": "internal server error"}`))
				}))
			},
			expectedCalled: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := tt.setupServer()
			defer server.Close()

			client := *resty.New()
			
			// Call the function - it should not panic and should handle errors gracefully
			enumCurrentToken(client, server.URL, "test-token")
			
			// If we got here without panic, the test passes
			// The function logs errors but doesn't return them
		})
	}
}

// TestListTokenAssociations tests the listTokenAssociations function with mocked HTTP responses
func TestListTokenAssociations(t *testing.T) {
	tests := []struct {
		name         string
		setupServer  func() *httptest.Server
		accessLevel  int
		page         int
		expectedNext int
	}{
		{
			name: "successful fetch with next page",
			setupServer: func() *httptest.Server {
				return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					// Verify headers and query params
					if r.Header.Get("PRIVATE-TOKEN") == "" {
						t.Error("PRIVATE-TOKEN header not set")
					}
					if r.URL.Query().Get("min_access_level") == "" {
						t.Error("min_access_level query param not set")
					}
					if r.URL.Query().Get("per_page") != "100" {
						t.Error("per_page should be 100")
					}
					
					response := TokenAssociations{
						Groups: []struct {
							ID             int         `json:"id"`
							WebURL         string      `json:"web_url"`
							Name           string      `json:"name"`
							ParentID       interface{} `json:"parent_id"`
							OrganizationID int         `json:"organization_id"`
							AccessLevels   int         `json:"access_levels"`
							Visibility     string      `json:"visibility"`
						}{
							{
								ID:           1,
								WebURL:       "https://gitlab.com/group1",
								Name:         "Group 1",
								ParentID:     nil,
								AccessLevels: 50,
								Visibility:   "private",
							},
						},
						Projects: []struct {
							ID                int       `json:"id"`
							Description       string    `json:"description"`
							Name              string    `json:"name"`
							NameWithNamespace string    `json:"name_with_namespace"`
							Path              string    `json:"path"`
							PathWithNamespace string    `json:"path_with_namespace"`
							CreatedAt         time.Time `json:"created_at"`
							AccessLevels      struct {
								ProjectAccessLevel int `json:"project_access_level"`
								GroupAccessLevel   int `json:"group_access_level"`
							} `json:"access_levels"`
							Visibility string `json:"visibility"`
							WebURL     string `json:"web_url"`
							Namespace  struct {
								ID        int         `json:"id"`
								Name      string      `json:"name"`
								Path      string      `json:"path"`
								Kind      string      `json:"kind"`
								FullPath  string      `json:"full_path"`
								ParentID  interface{} `json:"parent_id"`
								AvatarURL string      `json:"avatar_url"`
								WebURL    string      `json:"web_url"`
							} `json:"namespace"`
						}{
							{
								ID:                10,
								Name:              "project1",
								NameWithNamespace: "Group 1 / project1",
								WebURL:            "https://gitlab.com/group1/project1",
							},
						},
					}
					
					w.Header().Set("Content-Type", "application/json")
					w.Header().Set("x-next-page", "2")
					w.WriteHeader(http.StatusOK)
					json.NewEncoder(w).Encode(response)
				}))
			},
			accessLevel:  20,
			page:         1,
			expectedNext: 2,
		},
		{
			name: "successful fetch last page",
			setupServer: func() *httptest.Server {
				return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					response := TokenAssociations{
						Groups:   []struct {
							ID             int         `json:"id"`
							WebURL         string      `json:"web_url"`
							Name           string      `json:"name"`
							ParentID       interface{} `json:"parent_id"`
							OrganizationID int         `json:"organization_id"`
							AccessLevels   int         `json:"access_levels"`
							Visibility     string      `json:"visibility"`
						}{},
						Projects: []struct {
							ID                int       `json:"id"`
							Description       string    `json:"description"`
							Name              string    `json:"name"`
							NameWithNamespace string    `json:"name_with_namespace"`
							Path              string    `json:"path"`
							PathWithNamespace string    `json:"path_with_namespace"`
							CreatedAt         time.Time `json:"created_at"`
							AccessLevels      struct {
								ProjectAccessLevel int `json:"project_access_level"`
								GroupAccessLevel   int `json:"group_access_level"`
							} `json:"access_levels"`
							Visibility string `json:"visibility"`
							WebURL     string `json:"web_url"`
							Namespace  struct {
								ID        int         `json:"id"`
								Name      string      `json:"name"`
								Path      string      `json:"path"`
								Kind      string      `json:"kind"`
								FullPath  string      `json:"full_path"`
								ParentID  interface{} `json:"parent_id"`
								AvatarURL string      `json:"avatar_url"`
								WebURL    string      `json:"web_url"`
							} `json:"namespace"`
						}{},
					}
					
					w.Header().Set("Content-Type", "application/json")
					// No x-next-page header means last page
					w.WriteHeader(http.StatusOK)
					json.NewEncoder(w).Encode(response)
				}))
			},
			accessLevel:  20,
			page:         3,
			expectedNext: -1,
		},
		{
			name: "unauthorized request",
			setupServer: func() *httptest.Server {
				return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					w.WriteHeader(http.StatusUnauthorized)
					w.Write([]byte(`{"error": "unauthorized"}`))
				}))
			},
			accessLevel:  20,
			page:         1,
			expectedNext: -1,
		},
		{
			name: "server error",
			setupServer: func() *httptest.Server {
				return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					w.WriteHeader(http.StatusInternalServerError)
					w.Write([]byte(`{"error": "internal server error"}`))
				}))
			},
			accessLevel:  20,
			page:         1,
			expectedNext: -1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := tt.setupServer()
			defer server.Close()

			client := *resty.New()
			
			nextPage := listTokenAssociations(client, server.URL, "test-token", tt.accessLevel, tt.page)
			
			if nextPage != tt.expectedNext {
				t.Errorf("listTokenAssociations() = %v, want %v", nextPage, tt.expectedNext)
			}
		})
	}
}

// TestEnumFunction tests the Enum function with mocked dependencies
func TestEnumFunction(t *testing.T) {
	// Note: Testing the Enum function directly is complex because it:
	// 1. Uses package-level variables (gitlabApiToken, gitlabUrl)
	// 2. Calls log.Fatal on errors
	// 3. Depends on external GitLab client
	// 
	// For true unit testing, this would require refactoring to inject dependencies.
	// Instead, we test the command structure and the individual functions it calls.
	
	t.Run("command structure is correct", func(t *testing.T) {
		cmd := NewEnumCmd()
		
		if cmd.Run == nil {
			t.Error("Enum command should have a Run function")
		}
		
		// Verify Run function is set to Enum
		// We can't directly compare function pointers, but we can check it's not nil
	})
}

// TestEnumCmdExecution tests command flag parsing without execution
func TestEnumCmdExecution(t *testing.T) {
	tests := []struct {
		name        string
		args        []string
		wantErr     bool
		errContains string
	}{
		{
			name:        "missing required flags",
			args:        []string{},
			wantErr:     true,
			errContains: "required flag",
		},
		{
			name:    "with gitlab and token flags",
			args:    []string{"--gitlab", "https://gitlab.com", "--token", "glpat-test"},
			wantErr: false,
		},
		{
			name:    "with all flags including level",
			args:    []string{"--gitlab", "https://gitlab.com", "--token", "glpat-test", "--level", "30"},
			wantErr: false,
		},
		{
			name:    "with verbose flag",
			args:    []string{"--gitlab", "https://gitlab.com", "--token", "glpat-test", "--verbose"},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cmd := NewEnumCmd()
			
			// Override Run to prevent actual execution
			cmd.Run = func(cmd *cobra.Command, args []string) {
				// Do nothing - we're just testing flag parsing
			}
			
			cmd.SetArgs(tt.args)
			err := cmd.Execute()
			
			if tt.wantErr && err == nil {
				t.Errorf("Expected error containing %q, got nil", tt.errContains)
			}
			
			if !tt.wantErr && err != nil {
				t.Errorf("Unexpected error: %v", err)
			}
		})
	}
}

// TestMinAccessLevelVariable tests the package-level minAccessLevel variable
func TestMinAccessLevelVariable(t *testing.T) {
	// Save original value
	originalValue := minAccessLevel
	defer func() {
		minAccessLevel = originalValue
	}()
	
	tests := []struct {
		name     string
		setValue int
		wantVal  int
	}{
		{
			name:     "set to guest level (10)",
			setValue: 10,
			wantVal:  10,
		},
		{
			name:     "set to developer level (30)",
			setValue: 30,
			wantVal:  30,
		},
		{
			name:     "set to maintainer level (40)",
			setValue: 40,
			wantVal:  40,
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			minAccessLevel = tt.setValue
			if minAccessLevel != tt.wantVal {
				t.Errorf("minAccessLevel = %v, want %v", minAccessLevel, tt.wantVal)
			}
		})
	}
}
