package filters

import (
	"context"
	"encoding/json"
	"errors"
	"testing"

	"github.com/percybolmer/workflow/metric"
	"github.com/percybolmer/workflow/payload"
	"github.com/percybolmer/workflow/property"
)

func TestMapFilterHandle(t *testing.T) {
	type testCase struct {
		name            string
		expectedErr     error
		expectedPayload bool
		data            interface{}
		filters         map[string]string
		strict          bool
	}
	testCases := []testCase{
		{
			name:            "StrictModeWithBadValues",
			expectedErr:     nil,
			expectedPayload: false,
			strict:          true,
			data:            map[string]string{"hello": "world", "second": "wrong_value"},
			filters:         map[string]string{"hello": "world", "second": "notvalid"},
		}, {
			name:            "BadInputPayload",
			expectedErr:     ErrNotJSONMapInput,
			expectedPayload: false,
			strict:          true,
			data:            "NotAJSONMAP",
			filters:         map[string]string{"hello": "world", "second": "notvalid"},
		}, {
			name:            "GoodPayload",
			expectedErr:     nil,
			expectedPayload: true,
			strict:          false,
			data:            map[string]string{"hello": "world", "second": "wrong_value"},
			filters:         map[string]string{"hello": "world", "second": "notvalid"},
		}, {
			name:            "RegexpFilters",
			expectedErr:     nil,
			expectedPayload: true,
			strict:          false,
			data:            map[string]string{"hello": "world", "second": "wrong_value"},
			filters:         map[string]string{"hello": "wor.*"},
		}, {
			name:            "ComplexRegexpFilters",
			expectedErr:     nil,
			expectedPayload: true,
			strict:          false,
			data:            map[string]string{"hello": "world", "email": "thisisacool@email.com"},
			filters:         map[string]string{"email": "^[a-zA-Z0-9.!#$%&'*+\\/=?^_`{|}~-]+@[a-zA-Z0-9](?:[a-zA-Z0-9-]{0,61}[a-zA-Z0-9])?(?:\\.[a-zA-Z0-9](?:[a-zA-Z0-9-]{0,61}[a-zA-Z0-9])?)*$"},
		},
	}

	for _, tc := range testCases {
		rfg := NewMapFilterHandler()
		rfg.SetMetricProvider(metric.NewPrometheusProvider(), tc.name)
		rfg.Cfg.SetProperty("filters", tc.filters)
		rfg.Cfg.SetProperty("strict", tc.strict)
		_, miss := rfg.ValidateConfiguration()
		if miss != nil {
			t.Fatalf("%+v", miss)
		}
		data, err := json.Marshal(tc.data)
		if err != nil {
			t.Fatal(err)
		}
		pay := payload.BasePayload{
			Source:  "test",
			Payload: data,
		}

		err = rfg.Handle(context.Background(), pay)

		if !errors.Is(err, tc.expectedErr) {
			t.Fatalf("%s: %s : %s", tc.name, err, tc.expectedErr)
		}
		mets := rfg.metrics.GetMetric(rfg.MetricPayloadOut)
		if tc.expectedPayload && mets.Value == 0 {
			t.Fatalf("%s: test expects output from this Handler", tc.name)
		}

	}

}

func TestMapFilterValidateConfiguration(t *testing.T) {
	type testCase struct {
		Name        string
		Cfgs        map[string]interface{}
		IsValid     bool
		ExpectedErr error
	}

	testCases := []testCase{
		{Name: "InValidType", IsValid: false, Cfgs: map[string]interface{}{"strict": 1}, ExpectedErr: property.ErrWrongPropertyType},
		{Name: "NoSuchConfig", IsValid: false, Cfgs: map[string]interface{}{"ConfigThatDoesNotExist": true}, ExpectedErr: property.ErrNoSuchProperty},
		{Name: "MissingConfig", IsValid: false, Cfgs: nil, ExpectedErr: nil},
		{Name: "AssignedFilters", IsValid: true, Cfgs: map[string]interface{}{"filters": map[string]string{"thisfilter": "here"}}, ExpectedErr: nil},
	}

	for _, tc := range testCases {
		rfg := NewMapFilterHandler()

		for name, prop := range tc.Cfgs {
			err := rfg.Cfg.SetProperty(name, prop)
			if !errors.Is(err, tc.ExpectedErr) {
				if err != nil && tc.ExpectedErr != nil {
					t.Fatalf("%s: Expected: %s, but found: %s", tc.Name, tc.ExpectedErr, err.Error())
				}

			}
		}

		valid, _ := rfg.ValidateConfiguration()
		if !tc.IsValid && valid {
			t.Fatal("Missmatch between Valid and tc.IsValid")
		}

		if tc.Name == "AssignedFilters" {
			if rfg.filters["thisfilter"].String() != "here" {
				t.Fatal("Didnt find assigned filter")
			}
		}
	}
}
