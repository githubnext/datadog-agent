// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2016-present Datadog, Inc.

package transform

import (
	"testing"

	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/collector/pdata/pcommon"
)

func TestResourceContainerTagsLazyExtraction(t *testing.T) {
	resource := pcommon.NewResource()
	resource.Attributes().PutStr("container.id", "SHARED")
	resource.Attributes().PutStr("container.name", "My Container")
	resource.Attributes().PutStr("custom", "Value")
	resource.Attributes().PutInt("number", 42)
	for _, tc := range []struct {
		name string
		keys []string
		want []string
	}{
		{name: "nil"},
		{name: "empty", keys: []string{}},
		{name: "missing custom", keys: []string{"missing"}},
		{name: "missing mapped", keys: []string{"k8s.pod.name"}},
		{name: "custom only", keys: []string{"custom", "number", "missing"}, want: []string{"custom:value", "number:42"}},
		{name: "mapped only", keys: []string{"container.name", "container.id"}, want: []string{"container_name:my_container", "container_id:shared"}},
		{name: "mixed duplicates", keys: []string{"custom", "container.id", "custom", "container.name", "container.id"}, want: []string{"custom:value", "container_id:shared", "custom:value", "container_name:my_container", "container_id:shared"}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			require.Equal(t, tc.want, GetOTelContainerTags(resource.Attributes(), tc.keys))
		})
	}
	first := GetOTelContainerTags(resource.Attributes(), []string{"custom", "container.id"})
	secondResource := pcommon.NewResource()
	secondResource.Attributes().PutStr("container.id", "SHARED")
	secondResource.Attributes().PutStr("custom", "Other")
	require.Equal(t, []string{"custom:other", "container_id:shared"}, GetOTelContainerTags(secondResource.Attributes(), []string{"custom", "container.id"}))
	first[0] = "changed"
	require.Equal(t, []string{"custom:value", "container_id:shared"}, GetOTelContainerTags(resource.Attributes(), []string{"custom", "container.id"}))
}
