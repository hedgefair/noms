package enc

import (
	"encoding/json"
	"fmt"
	"sort"

	. "github.com/attic-labs/noms/dbg"
	"github.com/attic-labs/noms/ref"
	"github.com/attic-labs/noms/store"
	"github.com/attic-labs/noms/types"
)

var (
	jsonTag = []byte("j ")
)

func jsonEncode(v types.Value, s store.ChunkSink) (r ref.Ref, err error) {
	var j interface{}
	j, err = getJSON(v, s)
	if err != nil {
		return
	}
	w := s.Put()
	_, err = w.Write(jsonTag)
	if err == nil {
		err = json.NewEncoder(w).Encode(j)
	}
	if err != nil {
		return
	}
	return w.Ref()
}

func getJSON(v types.Value, s store.ChunkSink) (interface{}, error) {
	switch v := v.(type) {
	case types.Blob:
		Chk.Fail(fmt.Sprintf("jsonEncode doesn't support encoding blobs - didn't expect to get here: %+v", v))
	case types.Bool:
		return bool(v), nil
	case types.Float32:
		return map[string]interface{}{
			"float32": float32(v),
		}, nil
	case types.Float64:
		return map[string]interface{}{
			"float64": float64(v),
		}, nil
	case types.Int16:
		return map[string]interface{}{
			"int16": int16(v),
		}, nil
	case types.Int32:
		return map[string]interface{}{
			"int32": int32(v),
		}, nil
	case types.Int64:
		return map[string]interface{}{
			"int64": int64(v),
		}, nil
	case types.List:
		return getJSONList(v, s)
	case types.Map:
		return getJSONMap(v, s)
	case types.Set:
		return getJSONSet(v, s)
	case types.String:
		return v.String(), nil
	case types.UInt16:
		return map[string]interface{}{
			"uint16": uint16(v),
		}, nil
	case types.UInt32:
		return map[string]interface{}{
			"uint32": uint32(v),
		}, nil
	case types.UInt64:
		return map[string]interface{}{
			"uint64": uint64(v),
		}, nil
	default:
		panic("Unexpected type")
	}
	return nil, nil
}
func getJSONList(l types.List, s store.ChunkSink) (r interface{}, err error) {
	j := []interface{}{}
	for i := uint64(0); i < l.Len(); i++ {
		var cj interface{}
		cj, err = getChildJSON(l.Get(i), s)
		if err != nil {
			return
		}
		j = append(j, cj)
	}
	r = map[string]interface{}{
		"list": j,
	}
	return
}

func getJSONMap(m types.Map, s store.ChunkSink) (r interface{}, err error) {
	// Iteration through Set is random, but we need a deterministic order for serialization. Let's order using the refs of the values in the set.
	order := types.MapEntrySlice{}
	m.Iter(func(entry types.MapEntry) (stop bool) {
		order = append(order, entry)
		return
	})
	sort.Sort(order)

	j := []interface{}{}
	for _, r := range order {
		var cjk, cjv interface{}
		cjk, err = getChildJSON(r.Key, s)
		if err == nil {
			cjv, err = getChildJSON(r.Value, s)
		}
		if err != nil {
			return nil, err
		}
		j = append(j, cjk)
		j = append(j, cjv)
	}

	r = map[string]interface{}{
		"map": j,
	}
	return
}

func getJSONSet(set types.Set, s store.ChunkSink) (r interface{}, err error) {
	// Iteration through Set is random, but we need a deterministic order for serialization. Let's order using the refs of the values in the set.
	lookup := map[ref.Ref]types.Value{}
	order := ref.RefSlice{}
	set.Iter(func(v types.Value) (stop bool) {
		order = append(order, v.Ref())
		lookup[v.Ref()] = v
		return
	})
	sort.Sort(order)

	j := []interface{}{}
	for _, r := range order {
		v := lookup[r]
		var cj interface{}
		cj, err = getChildJSON(v, s)
		if err != nil {
			return nil, err
		}
		j = append(j, cj)
	}

	r = map[string]interface{}{
		"set": j,
	}
	return
}

func getChildJSON(v types.Value, s store.ChunkSink) (interface{}, error) {
	var r ref.Ref
	var err error
	switch v := v.(type) {
	// Blobs, lists, maps, and sets are always out-of-line
	case types.Blob:
		r, err = WriteValue(v, s)
	case types.List:
		r, err = WriteValue(v, s)
	case types.Map:
		r, err = WriteValue(v, s)
	case types.Set:
		r, err = WriteValue(v, s)
	default:
		// Other types are always inline.
		return getJSON(v, s)
	}
	if err != nil {
		return nil, err
	}
	return map[string]interface{}{
		"ref": r.String(),
	}, nil
}