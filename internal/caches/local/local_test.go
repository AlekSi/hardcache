package local

import (
	"encoding/hex"
	"errors"
	"io/fs"
	"log/slog"
	"math"
	"os"
	"path/filepath"
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

	c, err := cache.Open(dir)
	musta.NoError(t, err)

	t.Cleanup(func() {
		musta.NoError(t, c.Close())
	})

	actual, err := c.Get(actionID(t, "01a8b978c9044aabe4e554ee2d630f5437162fd385e60fbaf51492b4be15c226"))
	musta.NoError(t, err)

	expected := cache.Entry{
		OutputID: outputID(t, "e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855"),
		Size:     0,
		Time:     time.Date(2025, time.November, 17, 17, 13, 1, 426224000, time.UTC).Local(),
	}

	shoulda.BeDeepEqual(t, actual.Time, expected.Time)
	shoulda.BeDeepEqual(t, actual, expected)

	actual, err = c.Get(actionID(t, "fd8792322c0942921f5dca60ae1196d10c2195bfadef68f80f94de000625531c"))
	musta.NoError(t, err)

	expected.Time = time.Date(2025, time.November, 17, 17, 13, 2, 227666000, time.UTC).Local()

	shoulda.BeDeepEqual(t, actual.Time, expected.Time)
	shoulda.BeDeepEqual(t, actual, expected)

	// executable
	actual, err = c.Get(actionID(t, "b774285e0fffc3f1827be05e08ea22244e40ae09ca8359e45e329740aaa06dba"))
	musta.NoError(t, err)

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

	c, err := New(localtest.Setup(t), TrimLimits{}, logger(t))
	musta.NoError(t, err)

	before, freed, _ := c.TrimForce()
	shoulda.BeEqual(t, before, -1)
	shoulda.BeEqual(t, freed, 0)
}

func TestTrimCutoffNone(t *testing.T) {
	t.Parallel()

	c, err := New(localtest.Setup(t), TrimLimits{Cutoff: new(time.Date(2000, time.January, 1, 0, 0, 0, 0, time.UTC))}, logger(t))
	musta.NoError(t, err)

	before, freed, _ := c.TrimForce()
	shoulda.BeEqual(t, before, int64(109_518_524))
	shoulda.BeEqual(t, freed, int64(0))
}

func TestTrimCutoffAll(t *testing.T) {
	t.Parallel()

	c, err := New(localtest.Setup(t), TrimLimits{Cutoff: new(time.Date(2999, time.January, 1, 0, 0, 0, 0, time.UTC))}, logger(t))
	musta.NoError(t, err)

	before, freed, _ := c.TrimForce()
	shoulda.BeEqual(t, before, int64(109_518_524))
	shoulda.BeEqual(t, freed, int64(109_518_524))
}

func TestTrimCutoffPart(t *testing.T) {
	t.Parallel()

	c, err := New(localtest.Setup(t), TrimLimits{Cutoff: new(time.Date(2025, time.November, 17, 17, 13, 0, 0, time.UTC))}, logger(t))
	musta.NoError(t, err)

	before, freed, _ := c.TrimForce()
	shoulda.BeEqual(t, before, int64(109_518_524))
	shoulda.BeEqual(t, freed, int64(3_975_344))
}

func TestTrimSizeNone(t *testing.T) {
	t.Parallel()

	c, err := New(localtest.Setup(t), TrimLimits{MaxSize: new(int64(math.MaxInt64))}, logger(t))
	musta.NoError(t, err)

	before, freed, _ := c.TrimForce()
	shoulda.BeEqual(t, before, int64(109_518_524))
	shoulda.BeEqual(t, freed, int64(0))
}

func TestTrimSizeAll(t *testing.T) {
	t.Parallel()

	c, err := New(localtest.Setup(t), TrimLimits{MaxSize: new(int64(0))}, logger(t))
	musta.NoError(t, err)

	before, freed, _ := c.TrimForce()
	shoulda.BeEqual(t, before, int64(109_518_524))
	shoulda.BeEqual(t, freed, int64(109_518_524))
}

func TestTrimSizePart(t *testing.T) {
	t.Parallel()

	maxSize := int64(50_000_000)
	c, err := New(localtest.Setup(t), TrimLimits{MaxSize: new(maxSize)}, logger(t))
	musta.NoError(t, err)

	before, freed, _ := c.TrimForce()
	shoulda.BeEqual(t, before, int64(109_518_524))
	shoulda.BeEqual(t, freed, int64(60_023_595))
	shoulda.BeLess(t, before-freed, maxSize)
}

// entryExists returns true if the cache entry with the given name exists in dir.
func entryExists(t testing.TB, dir, name string) bool {
	t.Helper()

	_, err := os.Stat(filepath.Join(dir, name[:2], name))
	if errors.Is(err, fs.ErrNotExist) {
		return false
	}

	musta.NoError(t, err)

	return true
}

// Entries in testdata around the hot entries boundary used below, sorted by modification time.
const (
	lastColdEntry  = "2587e67cf130321e7701323bb0472af3f64ec5d6fbcb57faf2041e0ffbc9d6db-a" // 17:13:01.999457
	firstHotEntry  = "455313d9e42beccb1d3ceea9981fabaf275ef40caad25be314099bc9d7ae3333-d" // 17:13:02.005059
	ninthHotEntry  = "1e7102305cc1bed00af293b76a99362b4acadd0fd7f9579f235dda63d060303a-d" // 17:13:02.015730
	tenthHotEntry  = "3f528236779cfdf4970e27e45bd6cdc08de53e18e92fa567a83ce166ec95af1c-a" // 17:13:02.015882
	newestHotEntry = "b774285e0fffc3f1827be05e08ea22244e40ae09ca8359e45e329740aaa06dba-a" // 17:13:07.284400
)

// keepAfter makes 60_715_695 bytes of testdata entries hot.
var keepAfter = time.Date(2025, time.November, 17, 17, 13, 2, 0, time.UTC)

func TestTrimSizeKeepHot(t *testing.T) {
	t.Parallel()

	dir := localtest.Setup(t)

	maxSize := int64(50_000_000)
	c, err := New(dir, TrimLimits{MaxSize: new(maxSize), KeepAfter: new(keepAfter)}, logger(t))
	musta.NoError(t, err)

	// all cold entries are removed, all hot entries are kept
	before, freed, surge := c.TrimForce()
	shoulda.BeEqual(t, before, int64(109_518_524))
	shoulda.BeEqual(t, freed, int64(48_802_829))
	shoulda.BeEqual(t, surge, int64(10_715_695))
	shoulda.BeEqual(t, before-freed, maxSize+surge)

	shoulda.BeFalse(t, entryExists(t, dir, lastColdEntry))
	shoulda.BeTrue(t, entryExists(t, dir, firstHotEntry))
	shoulda.BeTrue(t, entryExists(t, dir, newestHotEntry))
}

func TestTrimSizeKeepHotAll(t *testing.T) {
	t.Parallel()

	dir := localtest.Setup(t)

	c, err := New(dir, TrimLimits{MaxSize: new(int64(0)), KeepAfter: new(keepAfter)}, logger(t))
	musta.NoError(t, err)

	// without a hard limit, all hot entries are kept
	before, freed, surge := c.TrimForce()
	shoulda.BeEqual(t, before, int64(109_518_524))
	shoulda.BeEqual(t, freed, int64(48_802_829))
	shoulda.BeEqual(t, surge, int64(60_715_695))

	shoulda.BeFalse(t, entryExists(t, dir, lastColdEntry))
	shoulda.BeTrue(t, entryExists(t, dir, firstHotEntry))
	shoulda.BeTrue(t, entryExists(t, dir, newestHotEntry))
}

func TestTrimSizeKeepHotUnderMaxSize(t *testing.T) {
	t.Parallel()

	c, err := New(localtest.Setup(t), TrimLimits{
		MaxSize:     new(int64(math.MaxInt64)),
		KeepAfter:   new(keepAfter),
		HardMaxSize: new(int64(math.MaxInt64)),
	}, logger(t))
	musta.NoError(t, err)

	before, freed, surge := c.TrimForce()
	shoulda.BeEqual(t, before, int64(109_518_524))
	shoulda.BeEqual(t, freed, int64(0))
	shoulda.BeEqual(t, surge, int64(0))
}

func TestTrimHardMaxSize(t *testing.T) {
	t.Parallel()

	dir := localtest.Setup(t)

	maxSize := int64(50_000_000)
	hardMaxSize := int64(55_000_000)
	c, err := New(dir, TrimLimits{
		MaxSize:     new(maxSize),
		KeepAfter:   new(keepAfter),
		HardMaxSize: new(hardMaxSize),
	}, logger(t))
	musta.NoError(t, err)

	// all cold entries and 9 least recently used hot entries are removed
	before, freed, surge := c.TrimForce()
	shoulda.BeEqual(t, before, int64(109_518_524))
	shoulda.BeEqual(t, freed, int64(54_938_320))
	shoulda.BeEqual(t, surge, int64(4_580_204))
	shoulda.BeLess(t, before-freed, hardMaxSize)
	shoulda.BeEqual(t, before-freed, maxSize+surge)

	shoulda.BeFalse(t, entryExists(t, dir, lastColdEntry))
	shoulda.BeFalse(t, entryExists(t, dir, firstHotEntry))
	shoulda.BeFalse(t, entryExists(t, dir, ninthHotEntry))
	shoulda.BeTrue(t, entryExists(t, dir, tenthHotEntry))
	shoulda.BeTrue(t, entryExists(t, dir, newestHotEntry))
}

func TestTrimHardMaxSizeOnly(t *testing.T) {
	t.Parallel()

	dir := localtest.Setup(t)

	// without max size, hard max size is a plain LRU limit, hot entries or not
	hardMaxSize := int64(50_000_000)
	c, err := New(dir, TrimLimits{KeepAfter: new(keepAfter), HardMaxSize: new(hardMaxSize)}, logger(t))
	musta.NoError(t, err)

	// the same as TestTrimSizePart
	before, freed, surge := c.TrimForce()
	shoulda.BeEqual(t, before, int64(109_518_524))
	shoulda.BeEqual(t, freed, int64(60_023_595))
	shoulda.BeEqual(t, surge, int64(0))
	shoulda.BeLess(t, before-freed, hardMaxSize)

	shoulda.BeFalse(t, entryExists(t, dir, firstHotEntry))
	shoulda.BeTrue(t, entryExists(t, dir, newestHotEntry))
}

func TestTrimKeepHotOnly(t *testing.T) {
	t.Parallel()

	// hot entries alone do not enable trimming
	c, err := New(localtest.Setup(t), TrimLimits{KeepAfter: new(keepAfter)}, logger(t))
	musta.NoError(t, err)

	before, freed, surge := c.TrimForce()
	shoulda.BeEqual(t, before, -1)
	shoulda.BeEqual(t, freed, 0)
	shoulda.BeEqual(t, surge, 0)
}
