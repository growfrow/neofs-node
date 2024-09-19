package shard_test

import (
	"bytes"
	"errors"
	"testing"
	"time"

	"github.com/nspcc-dev/neofs-node/pkg/core/object"
	"github.com/nspcc-dev/neofs-node/pkg/local_object_storage/shard"
	cidtest "github.com/nspcc-dev/neofs-sdk-go/container/id/test"
	objectSDK "github.com/nspcc-dev/neofs-sdk-go/object"
	oid "github.com/nspcc-dev/neofs-sdk-go/object/id"
	oidtest "github.com/nspcc-dev/neofs-sdk-go/object/id/test"
	"github.com/stretchr/testify/require"
)

func TestShard_Get(t *testing.T) {
	t.Run("without write cache", func(t *testing.T) {
		testShardGet(t, false)
	})

	t.Run("with write cache", func(t *testing.T) {
		testShardGet(t, true)
	})
}

func testShardGet(t *testing.T, hasWriteCache bool) {
	sh := newShard(t, hasWriteCache)
	defer releaseShard(sh, t)

	var putPrm shard.PutPrm
	var getPrm shard.GetPrm

	t.Run("small object", func(t *testing.T) {
		obj := generateObject()
		addAttribute(obj, "foo", "bar")
		addPayload(obj, 1<<5)
		addr := object.AddressOf(obj)

		putPrm.SetObject(obj)

		_, err := sh.Put(putPrm)
		require.NoError(t, err)

		getPrm.SetAddress(addr)

		res, err := testGet(t, sh, getPrm, hasWriteCache)
		require.NoError(t, err)
		require.Equal(t, obj, res.Object())
		require.True(t, res.HasMeta())

		testGetBytes(t, sh, addr, obj.Marshal())
	})

	t.Run("big object", func(t *testing.T) {
		obj := generateObject()
		addAttribute(obj, "foo", "bar")
		obj.SetID(oidtest.ID())
		addPayload(obj, 1<<20) // big obj
		addr := object.AddressOf(obj)

		putPrm.SetObject(obj)

		_, err := sh.Put(putPrm)
		require.NoError(t, err)

		getPrm.SetAddress(addr)

		res, err := testGet(t, sh, getPrm, hasWriteCache)
		require.NoError(t, err)
		require.Equal(t, obj, res.Object())
		require.True(t, res.HasMeta())

		testGetBytes(t, sh, addr, obj.Marshal())
	})

	t.Run("parent object", func(t *testing.T) {
		obj := generateObject()
		addAttribute(obj, "foo", "bar")
		cnr := cidtest.ID()
		splitID := objectSDK.NewSplitID()

		parent := generateObjectWithCID(cnr)
		addAttribute(parent, "parent", "attribute")

		child := generateObjectWithCID(cnr)
		child.SetParent(parent)
		idParent := parent.GetID()
		child.SetParentID(idParent)
		child.SetSplitID(splitID)
		addPayload(child, 1<<5)

		putPrm.SetObject(child)

		_, err := sh.Put(putPrm)
		require.NoError(t, err)

		getPrm.SetAddress(object.AddressOf(child))

		res, err := testGet(t, sh, getPrm, hasWriteCache)
		require.NoError(t, err)
		require.True(t, binaryEqual(child, res.Object()))

		getPrm.SetAddress(object.AddressOf(parent))

		_, err = testGet(t, sh, getPrm, hasWriteCache)

		var si *objectSDK.SplitInfoError
		require.True(t, errors.As(err, &si))

		link := si.SplitInfo().GetLink()
		require.True(t, link.IsZero())
		id1 := child.GetID()
		id2 := si.SplitInfo().GetLastPart()
		require.Equal(t, id1, id2)
		require.Equal(t, splitID, si.SplitInfo().SplitID())
	})
}

func testGet(t *testing.T, sh *shard.Shard, getPrm shard.GetPrm, hasWriteCache bool) (shard.GetRes, error) {
	res, err := sh.Get(getPrm)
	if hasWriteCache {
		require.Eventually(t, func() bool {
			if shard.IsErrNotFound(err) {
				res, err = sh.Get(getPrm)
			}
			return !shard.IsErrNotFound(err)
		}, time.Second, time.Millisecond*100)
	}
	return res, err
}

func testGetBytes(t testing.TB, sh *shard.Shard, addr oid.Address, objBin []byte) {
	b, err := sh.GetBytes(addr)
	require.NoError(t, err)
	require.Equal(t, objBin, b)

	b, hasMeta, err := sh.GetBytesWithMetadataLookup(addr)
	require.NoError(t, err)
	require.Equal(t, objBin, b)
	require.True(t, hasMeta)
}

// binary equal is used when object contains empty lists in the structure and
// requre.Equal fails on comparing <nil> and []{} lists.
func binaryEqual(a, b *objectSDK.Object) bool {
	return bytes.Equal(a.Marshal(), b.Marshal())
}
