package main

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"google.golang.org/protobuf/proto"
)

// TestSampleNestedReadonlyRoundTripPreservesData ensures SampleNested readonly wrappers
// round-trip without data loss and protect against external mutations.
func TestSampleNestedReadonlyRoundTripPreservesData(t *testing.T) {
	t.Parallel()

	// Arrange: Prepare a nested message with non-trivial payload.
	original := &SampleNested{
		NestedId:   123,
		NestedName: "alpha",
		NestedData: []byte("payload"),
	}
	snapshot := proto.Clone(original).(*SampleNested)

	// Act: Convert to readonly and mutate the original afterwards.
	readonly := original.ToReadonly()
	original.NestedId = 999
	original.NestedName = "beta"
	original.NestedData[0] = 'x'

	roundtrip := readonly.ToMessage()
	firstCopy := readonly.GetNestedData()
	firstCopy[0] = 'z'
	secondCopy := readonly.GetNestedData()

	// Assert: Readonly view matches the snapshot and mutations do not leak back.
	assert.Equal(t, snapshot.GetNestedId(), readonly.GetNestedId())
	assert.Equal(t, snapshot.GetNestedName(), readonly.GetNestedName())
	assert.Equal(t, snapshot.GetNestedData(), secondCopy)
	assert.True(t, proto.Equal(snapshot, roundtrip))
}

// TestSampleContainerReadonlyRoundTripPreservesCollections verifies nested, repeated,
// map, and oneof fields survive readonly conversions without data loss.
func TestSampleContainerReadonlyRoundTripPreservesCollections(t *testing.T) {
	t.Parallel()

	// Arrange: Build a container covering repeated/map/oneof branches.
	container := &SampleContainer{
		Id:    7,
		Title: "inventory",
		Nested: &SampleNested{
			NestedId:   555,
			NestedName: "inner",
			NestedData: []byte("seed"),
		},
		NestedList: []*SampleNested{
			{
				NestedId:   11,
				NestedName: "list-1",
			},
			nil,
		},
		NestedMap: map[string]*SampleNested{
			"first": {
				NestedId:   21,
				NestedName: "map-1",
			},
		},
		Payload: []byte{0x10, 0x20, 0x30},
		Labels:  []string{"daily", "weekly"},
	}
	container.OptionalValue = &SampleContainer_OptInt{OptInt: 77}
	snapshot := proto.Clone(container).(*SampleContainer)

	// Act: Convert to readonly and aggressively mutate the original instance.
	readonly := container.ToReadonly()
	container.Id = 100
	container.Title = "changed"
	container.Nested.NestedName = "mutated"
	container.NestedList[0].NestedId = 99
	container.NestedList = append(container.NestedList, &SampleNested{NestedId: 3})
	container.NestedMap["first"].NestedName = "altered"
	container.Payload[0] = 0xFF
	container.Labels[0] = "altered"
	container.OptionalValue = &SampleContainer_OptNested{OptNested: &SampleNested{NestedId: 50}}

	roundtrip := readonly.ToMessage()
	payloadCopy := readonly.GetPayload()
	if len(payloadCopy) > 0 {
		payloadCopy[0] ^= 0xFF
	}
	payloadCopyAfter := readonly.GetPayload()
	mapEntry := readonly.GetNestedMap()["first"]

	// Assert: Readonly data remains identical to the snapshot and roundtrip matches.
	assert.Equal(t, snapshot.GetId(), readonly.GetId())
	assert.Equal(t, snapshot.GetTitle(), readonly.GetTitle())
	assert.Equal(t, snapshot.GetNested().GetNestedName(), readonly.GetNested().GetNestedName())
	assert.Len(t, readonly.GetNestedList(), len(snapshot.GetNestedList()))
	for idx, wantItem := range snapshot.GetNestedList() {
		readonlyItem := readonly.GetNestedList()[idx]
		if wantItem == nil {
			assert.Nil(t, readonlyItem)
			continue
		}
		if assert.NotNil(t, readonlyItem) {
			assert.Equal(t, wantItem.GetNestedId(), readonlyItem.GetNestedId())
			assert.Equal(t, wantItem.GetNestedName(), readonlyItem.GetNestedName())
		}
	}
	assert.Equal(t, snapshot.GetNestedMap()["first"].GetNestedName(), mapEntry.GetNestedName())
	assert.Equal(t, snapshot.GetLabels(), readonly.GetLabels())
	assert.Equal(t, snapshot.GetPayload(), payloadCopyAfter)
	assert.Equal(t, SampleContainer_EnOptionalValueID_OptInt, readonly.GetOptionalValueOneofCase())
	assert.Equal(t, GetReflectTypeSampleContainer_OptInt(), readonly.GetOptionalValueReflectType())
	assert.True(t, proto.Equal(snapshot, roundtrip))
}

// TestSampleContainerReadonlyOptionalNestedCase checks optional nested case and clone behavior.
func TestSampleContainerReadonlyOptionalNestedCase(t *testing.T) {
	t.Parallel()

	// Arrange: Only populate the oneof with the nested branch and leave other fields nil.
	original := &SampleContainer{
		Id: 1,
		Nested: &SampleNested{
			NestedId:   100,
			NestedName: "primary",
		},
		OptionalValue: &SampleContainer_OptNested{
			OptNested: &SampleNested{
				NestedId:   200,
				NestedName: "opt",
			},
		},
	}

	// Act: Convert to readonly, clone back, and mutate the clone.
	readonly := original.ToReadonly()
	cloned := readonly.CloneMessage()
	assert.True(t, proto.Equal(original, cloned))
	cloned.Nested.NestedName = "mutated"
	if opt := cloned.GetOptNested(); opt != nil {
		opt.NestedName = "mutated-opt"
	}

	// Assert: Optional metadata and nested data stay intact in the readonly wrapper.
	assert.Equal(t, SampleContainer_EnOptionalValueID_OptNested, readonly.GetOptionalValueOneofCase())
	assert.Equal(t, GetReflectTypeSampleContainer_OptNested(), readonly.GetOptionalValueReflectType())
	if assert.NotNil(t, readonly.GetOptNested()) {
		assert.Equal(t, original.GetOptNested().GetNestedId(), readonly.GetOptNested().GetNestedId())
	}
	assert.Nil(t, readonly.GetNestedMap())
	assert.Equal(t, "primary", readonly.GetNested().GetNestedName())
	assert.NotEqual(t, readonly.GetNested().GetNestedName(), cloned.GetNested().GetNestedName())
	assert.Equal(t, "opt", readonly.GetOptNested().GetNestedName())
}

// TestSampleContainerReadonlyNilSafe ensures nil receivers don't panic when calling ToReadonly.
func TestSampleContainerReadonlyNilSafe(t *testing.T) {
	t.Parallel()

	// Arrange: A nil container reference.
	var container *SampleContainer

	// Act & Assert: Calling ToReadonly on nil should simply return nil.
	assert.Nil(t, container.ToReadonly())
}
