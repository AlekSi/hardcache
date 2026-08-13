package local

import (
	"encoding/hex"
	"log/slog"
	"math"
	"testing"
	"time"

	"github.com/AlekSi/shoulda"
	"github.com/AlekSi/shoulda/musta"

	"github.com/AlekSi/hardcache/internal/caches/local/localtest"
	"github.com/AlekSi/hardcache/internal/go/cache"
)

// logger returns a [slog.Logger] for the given test.
func logger(t testing.TB) *slog.Logger {
	t.Helper()

	opts := &slog.HandlerOptions{
		AddSource: true,
		Level:     slog.LevelDebug,
	}
	return slog.New(slog.NewTextHandler(t.Output(), opts))
}

// fromHex decodes a hex string to a byte array.
func fromHex(t testing.TB, s string) [cache.HashSize]byte {
	t.Helper()

	var res [cache.HashSize]byte
	n, err := hex.Decode(res[:], []byte(s))
	musta.NoError(t, err)
	musta.BeEqual(t, n, cache.HashSize)

	return res
}

// actionID converts a hex string to an ActionID.
func actionID(t testing.TB, s string) cache.ActionID {
	t.Helper()
	return cache.ActionID(fromHex(t, s))
}

// outputID converts a hex string to an OutputID.
func outputID(t testing.TB, s string) cache.OutputID {
	t.Helper()
	return cache.OutputID(fromHex(t, s))
}

func TestCache(t *testing.T) {
	t.Parallel()

	dir := localtest.Setup(t)

	c := musta.NotFail(cache.Open(dir))(t)

	t.Cleanup(func() {
		musta.NoError(t, c.Close())
	})

	actual := musta.NotFail(c.Get(actionID(t, "01a8b978c9044aabe4e554ee2d630f5437162fd385e60fbaf51492b4be15c226")))(t)

	expected := cache.Entry{
		OutputID: outputID(t, "e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855"),
		Size:     0,
		Time:     time.Date(2025, time.November, 17, 17, 13, 1, 426224000, time.UTC).Local(),
	}

	shoulda.BeDeepEqual(t, actual.Time, expected.Time)
	shoulda.BeDeepEqual(t, actual, expected)

	actual = musta.NotFail(c.Get(actionID(t, "fd8792322c0942921f5dca60ae1196d10c2195bfadef68f80f94de000625531c")))(t)

	expected.Time = time.Date(2025, time.November, 17, 17, 13, 2, 227666000, time.UTC).Local()

	shoulda.BeDeepEqual(t, actual.Time, expected.Time)
	shoulda.BeDeepEqual(t, actual, expected)

	// executable
	actual = musta.NotFail(c.Get(actionID(t, "b774285e0fffc3f1827be05e08ea22244e40ae09ca8359e45e329740aaa06dba")))(t)

	expected = cache.Entry{
		OutputID: outputID(t, "4b949c7e306fe19cf063f3dc5c1ab963a09a6bea4f7545974fe7385bdbaace94"),
		Size:     1060642,
		Time:     time.Date(2025, time.November, 17, 17, 13, 7, 284280000, time.UTC).Local(),
	}

	shoulda.BeDeepEqual(t, actual.Time, expected.Time)
	shoulda.BeDeepEqual(t, actual, expected)
}

func TestTrimNoop(t *testing.T) {
	t.Parallel()

	c, err := New(localtest.Setup(t), nil, nil, logger(t))
	musta.NoError(t, err)

	before, freed := c.TrimForce()
	shoulda.BeEqual(t, before, -1)
	shoulda.BeEqual(t, freed, 0)
}

func TestTrimCutoffNone(t *testing.T) {
	t.Parallel()

	cutoff := time.Date(2000, time.January, 1, 0, 0, 0, 0, time.UTC)
	c := musta.NotFail(New(localtest.Setup(t), &cutoff, nil, logger(t)))(t)

	before, freed := c.TrimForce()
	shoulda.BeEqual(t, before, int64(109_518_524))
	shoulda.BeEqual(t, freed, int64(0))
}

func TestTrimCutoffAll(t *testing.T) {
	t.Parallel()

	cutoff := time.Date(2999, time.January, 1, 0, 0, 0, 0, time.UTC)
	c := musta.NotFail(New(localtest.Setup(t), &cutoff, nil, logger(t)))(t)

	before, freed := c.TrimForce()
	shoulda.BeEqual(t, before, int64(109_518_524))
	shoulda.BeEqual(t, freed, int64(109_518_524))
}

func TestTrimCutoffPart(t *testing.T) {
	t.Parallel()

	cutoff := time.Date(2025, time.November, 17, 17, 13, 0, 0, time.UTC)
	c := musta.NotFail(New(localtest.Setup(t), &cutoff, nil, logger(t)))(t)

	before, freed := c.TrimForce()
	shoulda.BeEqual(t, before, int64(109_518_524))
	shoulda.BeEqual(t, freed, int64(3_975_344))
}

func TestTrimSizeNone(t *testing.T) {
	t.Parallel()

	maxSize := int64(math.MaxInt64)
	c := musta.NotFail(New(localtest.Setup(t), nil, &maxSize, logger(t)))(t)

	before, freed := c.TrimForce()
	shoulda.BeEqual(t, before, int64(109_518_524))
	shoulda.BeEqual(t, freed, int64(0))
}

func TestTrimSizeAll(t *testing.T) {
	t.Parallel()

	maxSize := int64(0)
	c := musta.NotFail(New(localtest.Setup(t), nil, &maxSize, logger(t)))(t)

	before, freed := c.TrimForce()
	shoulda.BeEqual(t, before, int64(109_518_524))
	shoulda.BeEqual(t, freed, int64(109_518_524))
}

func TestTrimSizePart(t *testing.T) {
	t.Parallel()

	maxSize := int64(50_000_000)
	c := musta.NotFail(New(localtest.Setup(t), nil, &maxSize, logger(t)))(t)

	before, freed := c.TrimForce()
	shoulda.BeEqual(t, before, int64(109_518_524))
	shoulda.BeEqual(t, freed, int64(60_023_595))
	shoulda.BeLess(t, before-freed, maxSize)
}

func TestStatusFixture(t *testing.T) {
	t.Parallel()

	c := musta.NotFail(New(localtest.Setup(t), nil, nil, logger(t)))(t)

	stats := c.Status()
	shoulda.BeEqual(t, stats.Entries, 1219)
	shoulda.BeEqual(t, stats.Bytes, int64(109_518_524))
	musta.NotBeZero(t, stats.Oldest)
	musta.NotBeZero(t, stats.Newest)
	shoulda.CompareLess(t, *stats.Oldest, *stats.Newest, time.Time.Compare)
}

func TestStatusEmpty(t *testing.T) {
	t.Parallel()

	maxSize := int64(0)
	c := musta.NotFail(New(localtest.Setup(t), nil, &maxSize, logger(t)))(t)
	before, freed := c.TrimForce()
	shoulda.BeEqual(t, before, int64(109_518_524))
	shoulda.BeEqual(t, freed, before)

	stats := c.Status()
	shoulda.BeEqual(t, stats.Entries, 0)
	shoulda.BeEqual(t, stats.Bytes, int64(0))
	shoulda.BeZero(t, stats.Oldest)
	shoulda.BeZero(t, stats.Newest)
}
