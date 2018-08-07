// ds is a package for using Google Cloud Datastore.
package ds

import (
	"context"
	"fmt"
	"time"

	"cloud.google.com/go/datastore"
	"golang.org/x/sync/errgroup"
	"google.golang.org/api/iterator"
	"google.golang.org/api/option"

	"go.skia.org/infra/go/auth"
	"go.skia.org/infra/go/sklog"
	"go.skia.org/infra/go/util"
)

type Kind string

// Below are all the Kinds used in all applications. New Kinds should be listed
// here, and they should all have unique values, because when defining indexes
// for Cloud Datastore the index config is per project, not pre-namespace.
// Remember to add them to KindsToBackup if they are to be backed up, and push
// a new version of /ds/go/datastore_backup.
const (
	// Predict
	FAILURES     Kind = "Failures"
	FLAKY_RANGES Kind = "FlakyRanges"

	// Perf
	SHORTCUT   Kind = "Shortcut"
	ACTIVITY   Kind = "Activity"
	REGRESSION Kind = "Regression"
	ALERT      Kind = "Alert"

	// Gold
	ISSUE                  Kind = "Issue"
	TRYJOB                 Kind = "Tryjob"
	TRYJOB_RESULT          Kind = "TryjobResult"
	TRYJOB_EXP_CHANGE      Kind = "TryjobExpChange"
	TEST_DIGEST_EXP        Kind = "TryjobTestDigestExp" // TODO(stephana): Remove after migration to consolidated expectations store
	TRYJOB_TEST_DIGEST_EXP Kind = "TryjobTestDigestExp"
	MASTER_EXP_CHANGE      Kind = "MasterExpChange"
	IGNORE_RULE            Kind = "IgnoreRule"
	HELPER_RECENT_KEYS     Kind = "HelperRecentKeys"
	EXPECTATIONS_BLOB      Kind = "ExpectationsBlob"
	EXPECTATIONS_BLOB_ROOT Kind = "ExpectationsBlobRoot"

	// Android Compile
	COMPILE_TASK Kind = "CompileTask"

	// Leasing
	TASK Kind = "Task"

	// CT
	CAPTURE_SKPS_TASKS              Kind = "CaptureSkpsTasks"
	CHROMIUM_ANALYSIS_TASKS         Kind = "ChromiumAnalysisTasks"
	CHROMIUM_BUILD_TASKS            Kind = "ChromiumBuildTasks"
	CHROMIUM_PERF_TASKS             Kind = "ChromiumPerfTasks"
	LUA_SCRIPT_TASKS                Kind = "LuaScriptTasks"
	METRICS_ANALYSIS_TASKS          Kind = "MetricsAnalysisTasks"
	PIXEL_DIFF_TASKS                Kind = "PixelDiffTasks"
	RECREATE_PAGESETS_TASKS         Kind = "RecreatePageSetsTasks"
	RECREATE_WEBPAGE_ARCHIVES_TASKS Kind = "RecreateWebpageArchivesTasks"
	CLUSTER_TELEMETRY_IDS           Kind = "ClusterTelemetryIDs"

	// Autoroll
	KIND_AUTOROLL_MODE              Kind = "AutorollMode"
	KIND_AUTOROLL_MODE_ANCESTOR     Kind = "AutorollModeAncestor" // Fake; used to force strong consistency for testing's sake.
	KIND_AUTOROLL_ROLL              Kind = "AutorollRoll"
	KIND_AUTOROLL_ROLL_ANCESTOR     Kind = "AutorollRollAncestor" // Fake; used to force strong consistency for testing's sake.
	KIND_AUTOROLL_STATUS            Kind = "AutorollStatus"
	KIND_AUTOROLL_STATUS_ANCESTOR   Kind = "AutorollStatusAncestor" // Fake; used to force strong consistency for testing's sake.
	KIND_AUTOROLL_STRATEGY          Kind = "AutorollStrategy"
	KIND_AUTOROLL_STRATEGY_ANCESTOR Kind = "AutorollStrategyAncestor" // Fake; used to force strong consistency for testing's sake.
)

// Namespaces that are used in production, and thus might be backed up.
const (
	// Perf
	PERF_NS                = "perf"
	PERF_ANDROID_NS        = "perf-android"
	PERF_ANDROID_MASTER_NS = "perf-androidmaster"

	// Gold
	GOLD_SKIA_PROD_NS = "gold-skia-prod"

	// Android Compile
	ANDROID_COMPILE_NS = "android-compile"

	// Leasing
	LEASING_SERVER_NS = "leasing-server"

	// CT
	CT_NS = "cluster-telemetry"

	// Autoroll
	AUTOROLL_NS = "autoroll"
)

var (
	// KindsToBackup is a map from namespace to the list of Kinds to backup.
	// If this value is changed then remember to push a new version of /ds/go/datastore_backup.
	KindsToBackup = map[string][]Kind{
		AUTOROLL_NS:            []Kind{KIND_AUTOROLL_MODE, KIND_AUTOROLL_MODE_ANCESTOR, KIND_AUTOROLL_ROLL, KIND_AUTOROLL_ROLL_ANCESTOR, KIND_AUTOROLL_STATUS, KIND_AUTOROLL_STATUS_ANCESTOR, KIND_AUTOROLL_STRATEGY, KIND_AUTOROLL_STRATEGY_ANCESTOR},
		PERF_NS:                []Kind{ACTIVITY, ALERT, REGRESSION, SHORTCUT},
		PERF_ANDROID_NS:        []Kind{ACTIVITY, ALERT, REGRESSION, SHORTCUT},
		PERF_ANDROID_MASTER_NS: []Kind{ACTIVITY, ALERT, REGRESSION, SHORTCUT},
		GOLD_SKIA_PROD_NS:      []Kind{ISSUE, TRYJOB, TRYJOB_RESULT, TRYJOB_EXP_CHANGE, TEST_DIGEST_EXP, TRYJOB_TEST_DIGEST_EXP, MASTER_EXP_CHANGE, IGNORE_RULE, HELPER_RECENT_KEYS, EXPECTATIONS_BLOB, EXPECTATIONS_BLOB_ROOT},
		ANDROID_COMPILE_NS:     []Kind{COMPILE_TASK},
		LEASING_SERVER_NS:      []Kind{TASK},
		CT_NS:                  []Kind{CAPTURE_SKPS_TASKS, CHROMIUM_ANALYSIS_TASKS, CHROMIUM_BUILD_TASKS, CHROMIUM_PERF_TASKS, LUA_SCRIPT_TASKS, METRICS_ANALYSIS_TASKS, PIXEL_DIFF_TASKS, RECREATE_PAGESETS_TASKS, RECREATE_WEBPAGE_ARCHIVES_TASKS, CLUSTER_TELEMETRY_IDS},
	}
)

var (
	// DS is the Cloud Datastore client. Valid after Init() has been called.
	DS *datastore.Client

	// Namespace is the datastore namespace that data will be stored in. Valid after Init() has been called.
	Namespace string
)

// InitWithOpt the Cloud Datastore Client (DS).
//
// project - The project name, i.e. "google.com:skia-buildbots".
// ns      - The datastore namespace to store data into.
// opt     - Options to pass to the client.
func InitWithOpt(project string, ns string, opts ...option.ClientOption) error {
	if ns == "" {
		return sklog.FmtErrorf("Datastore namespace cannot be empty.")
	}

	Namespace = ns
	var err error
	DS, err = datastore.NewClient(context.Background(), project, opts...)
	if err != nil {
		return fmt.Errorf("Failed to initialize Cloud Datastore: %s", err)
	}
	return nil
}

// Init the Cloud Datastore Client (DS).
//
// project - The project name, i.e. "google.com:skia-buildbots".
// ns      - The datastore namespace to store data into.
func Init(project string, ns string) error {
	tok, err := auth.NewDefaultJWTServiceAccountTokenSource("https://www.googleapis.com/auth/datastore")
	if err != nil {
		return err
	}
	return InitWithOpt(project, ns, option.WithTokenSource(tok))
}

// InitForTesting is an init to call when running tests. It doesn't do any
// auth as it is expecting to run against the Cloud Datastore Emulator.
// See https://cloud.google.com/datastore/docs/tools/datastore-emulator
//
// project - The project name, i.e. "google.com:skia-buildbots".
// ns      - The datastore namespace to store data into.
func InitForTesting(project string, ns string, kinds ...Kind) error {
	Namespace = ns
	var err error
	DS, err = datastore.NewClient(context.Background(), project)
	if err != nil {
		return fmt.Errorf("Failed to initialize Cloud Datastore: %s", err)
	}
	return nil
}

// DeleteAll removes all entities of the given kind. If wait is true it waits
// until an eventually consistent query of the Kind returns a count of 0.
// Upon success the number of deleted entities is returned.
//
// Note: This is a very expensive operation if there are many entities of this
// kind and should be run as an 'offline' task.
func DeleteAll(client *datastore.Client, kind Kind, wait bool) (int, error) {
	const (
		// keyPageSize is the number of keys we retrieve at once
		keyPageSize = 10000

		// At most 500 keys can be deleted at once (Cloud Datastore limitation)
		deletePageSize = 500
	)

	sliceIter := newKeySliceIterator(client, kind, keyPageSize)
	slice, done, err := sliceIter.next()
	ctx := context.TODO()
	keySlices := [][]*datastore.Key{}

	totalKeyCount := 0
	for !done && (err == nil) {
		keySlices = append(keySlices, slice)
		totalKeyCount += len(slice)
		slice, done, err = sliceIter.next()
		sklog.Infof("Loaded %s %d keys %d", kind, len(slice), totalKeyCount)
	}
	if err != nil {
		return 0, err
	}

	// Delete all slices in parallel.
	var egroup errgroup.Group
	for _, slice := range keySlices {
		func(slice []*datastore.Key) {
			egroup.Go(func() error {
				for len(slice) > 0 {
					targetSlice := slice[:util.MinInt(deletePageSize, len(slice))]
					if err := client.DeleteMulti(ctx, targetSlice); err != nil {
						return err
					}
					slice = slice[len(targetSlice):]
				}
				return nil
			})
		}(slice)
	}
	if err := egroup.Wait(); err != nil {
		return 0, sklog.FmtErrorf("Error deleting entities: %s", err)
	}

	// If we need to wait loop until the entity count goes to zero.
	if wait {
		found := 1
		for found > 0 {
			if found, err = client.Count(context.TODO(), NewQuery(kind)); err != nil {
				return 0, err
			}
			// Sleep proportional to the number of found keys, but no more than 10 seconds.
			sleepTimeMs := util.MinInt64(int64(found)*10, 10000)
			time.Sleep(time.Duration(sleepTimeMs) * time.Millisecond)
		}
	}
	return totalKeyCount, nil
}

// Creates a new indeterminate key of the given kind.
func NewKey(kind Kind) *datastore.Key {
	return &datastore.Key{
		Kind:      string(kind),
		Namespace: Namespace,
	}
}

func NewKeyWithParent(kind Kind, parent *datastore.Key) *datastore.Key {
	ret := NewKey(kind)
	ret.Parent = parent
	return ret
}

// Creates a new query of the given kind with the right namespace.
func NewQuery(kind Kind) *datastore.Query {
	return datastore.NewQuery(string(kind)).Namespace(Namespace)
}

// IterKeysItem is the item returned by the IterKeys function via a channel.
type IterKeysItem struct {
	Keys []*datastore.Key
	Err  error
}

// IterKeys iterates all keys of the specified kind in slices of pageSize length.
func IterKeys(client *datastore.Client, kind Kind, pageSize int) (<-chan *IterKeysItem, error) {
	sliceIter := newKeySliceIterator(client, kind, pageSize)
	keySlice, done, err := sliceIter.next()
	if err != nil {
		return nil, err
	}

	retCh := make(chan *IterKeysItem)
	go func() {
		defer close(retCh)

		keyCount := 0
		for !done {
			keyCount += len(keySlice)
			retCh <- &IterKeysItem{
				Keys: keySlice,
				Err:  err,
			}
			// Get the next slice of keys.
			keySlice, done, err = sliceIter.next()
		}
	}()
	return retCh, nil
}

// keySliceIterator allows to iterate over the keys of a specific entity
// in slices of fixed size.
type keySliceIterator struct {
	client    *datastore.Client
	kind      Kind
	pageSize  int
	orderedBy []string
	cursorStr string
	done      bool
}

// newKeySliceIterator returns a new keySliceIterator instance for the given kind.
// 'pageSize' defines the size of slices that are returned by the next method.
// 'orderedBy' allows to sort the slices with the same operators as datastore.Query.
func newKeySliceIterator(client *datastore.Client, kind Kind, pageSize int, orderedBy ...string) *keySliceIterator {
	return &keySliceIterator{
		client:    client,
		kind:      kind,
		pageSize:  pageSize,
		orderedBy: orderedBy,
		cursorStr: "",
	}
}

// next returns the next slice of keys of the iterator. If the returned bool is
// true no more keys are available.
func (k *keySliceIterator) next() ([]*datastore.Key, bool, error) {
	// Once we have reached the end, don't run the query again.
	if k.done {
		return []*datastore.Key{}, true, nil
	}

	query := NewQuery(k.kind).KeysOnly().Limit(k.pageSize)
	for _, ob := range k.orderedBy {
		query = query.Order(ob)
	}

	if k.cursorStr != "" {
		cursor, err := datastore.DecodeCursor(k.cursorStr)
		if err != nil {
			return nil, false, sklog.FmtErrorf("Bad cursor %s: %s", k.cursorStr, err)
		}
		query = query.Start(cursor)
	}

	it := k.client.Run(context.TODO(), query)
	var err error
	var key *datastore.Key
	retKeys := make([]*datastore.Key, 0, k.pageSize)

	for {
		if key, err = it.Next(nil); err != nil {
			break
		}
		retKeys = append(retKeys, key)
	}

	if err != iterator.Done {
		return nil, false, sklog.FmtErrorf("Error retrieving keys: %s", err)
	}

	// Get the string for the next page.
	cursor, err := it.Cursor()
	if err != nil {
		return nil, false, sklog.FmtErrorf("Error retrieving next cursor: %s", err)
	}

	// Check if the string representation of the cursor has changed.
	newCursorStr := cursor.String()
	k.done = (k.cursorStr == newCursorStr)
	k.cursorStr = newCursorStr

	// We are not officially done while we have results to return.
	return retKeys, !(len(retKeys) > 0), nil
}
