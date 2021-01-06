package sql

import (
	"crypto/md5"
	"encoding/hex"
	"strconv"
	"strings"

	"go.skia.org/infra/go/jsonutils"
	"go.skia.org/infra/go/skerr"
	"go.skia.org/infra/golden/go/sql/schema"
	"go.skia.org/infra/golden/go/types"
)

// DigestToBytes returns the given digest as bytes. It returns an error if a missing digest is
// passed in or if the bytes are otherwise invalid.
func DigestToBytes(d types.Digest) (schema.DigestBytes, error) {
	if len(d) != 32 {
		return schema.DigestBytes{}, skerr.Fmt("Empty/missing/invalid digest passed in %q", d)
	}
	out := make(schema.DigestBytes, md5.Size)
	_, err := hex.Decode(out, []byte(d))
	return out, skerr.Wrap(err)
}

// SerializeMap returns the given map in JSON and the MD5 of that json string. nil maps will be
// treated as empty maps. Keys will be sorted lexicographically.
func SerializeMap(m map[string]string) (string, []byte) {
	var jsonBytes []byte
	if len(m) == 0 {
		jsonBytes = []byte("{}")
	} else {
		jsonBytes = jsonutils.MarshalStringMap(m)
	}

	h := md5.Sum(jsonBytes)
	return string(jsonBytes), h[:]
}

// TraceValuesShards is the number of shards we use in the TraceValues Table.
const TraceValuesShards = 8

// ComputeTraceValueShard computes a shard in the range [0,TraceValuesShards) based off the given
// trace hash bytes. This shard range was chosen to be a small number (relative to the thousands
// of ranges that the TraceValues database occupies) that is larger than the number of nodes in the
// cockroachdb cluster. See the overall design doc (go/skia-gold-sql) for more explanation about
// data locality and sharding.
func ComputeTraceValueShard(traceID schema.TraceID) byte {
	return traceID[0] % TraceValuesShards
}

// ComputeTileStartID returns the commit id is the beginning of the tile that the commit is in.
func ComputeTileStartID(cid schema.CommitID, tileWidth int) schema.CommitID {
	return (cid / schema.CommitID(tileWidth)) * schema.CommitID(tileWidth)
}

// AsMD5Hash returns the given byte slice as an MD5Hash (for easier use with maps)
func AsMD5Hash(b []byte) schema.MD5Hash {
	var m schema.MD5Hash
	copy(m[:], b)
	return m
}

// ValuesPlaceholders returns a set of SQL placeholder numbers grouped for use in an INSERT
// statement. For example, ValuesPlaceholders(2,3) returns ($1, $2), ($3, $4), ($5, $6)
// It panics if either param is <= 0.
func ValuesPlaceholders(valuesPerRow, numRows int) string {
	if valuesPerRow <= 0 || numRows <= 0 {
		panic("Cannot make ValuesPlaceholder with 0 rows or 0 values per row")
	}
	values := strings.Builder{}
	// There are at most 5 bytes per value that need to be written
	values.Grow(5 * valuesPerRow * numRows)
	// All WriteString calls below return nil errors, as specified in the documentation of
	// strings.Builder, so it is safe to ignore them.
	for argIdx := 1; argIdx <= valuesPerRow*numRows; argIdx += valuesPerRow {
		if argIdx != 1 {
			_, _ = values.WriteString(",")
		}
		_, _ = values.WriteString("(")
		for i := 0; i < valuesPerRow; i++ {
			if i != 0 {
				_, _ = values.WriteString(",")
			}
			_, _ = values.WriteString("$")
			_, _ = values.WriteString(strconv.Itoa(argIdx + i))
		}
		_, _ = values.WriteString(")")
	}
	return values.String()
}
