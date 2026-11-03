package store

import "crypto/sha256"

// Exact source SQL; changing even whitespace invalidates the LOCAL catalog hash.
const (
	readFeedsSQL          = `SELECT id,kind,CASE WHEN octet_length(title)<=4096 THEN title ELSE NULL END FROM public.app_prosody_feeds WHERE workspace_id=$1 AND deleted_at IS NULL AND id>$2::uuid ORDER BY id LIMIT $3`
	readClockSQL          = `SELECT pg_catalog.clock_timestamp()`
	readReadyBriefingsSQL = `SELECT s.id,r.id,r.published_at,
 CASE WHEN octet_length(s.source_bytes)<=65536 THEN s.source_bytes ELSE NULL END,
 CASE WHEN octet_length(r.script_bytes)<=256000 THEN r.script_bytes ELSE NULL END,
 CASE WHEN octet_length(r.manifest_bytes)<=262144 THEN r.manifest_bytes ELSE NULL END,
 (SELECT count(*) FROM public.app_prosody_renditions all_r WHERE all_r.workspace_id=s.workspace_id AND all_r.item_id=s.id)
 FROM public.app_prosody_submissions s
 JOIN public.app_prosody_renditions r ON r.workspace_id=s.workspace_id AND r.item_id=s.id
 WHERE s.workspace_id=$1 AND s.feed_id=$2 AND s.status='ready' AND s.deleted_at IS NULL
 AND r.published_at<=$3
 AND ($4::boolean=false OR (r.published_at,s.id,r.id)>($5::timestamptz,$6::uuid,$7::uuid))
 ORDER BY r.published_at,s.id,r.id LIMIT $8 FOR SHARE OF s`
	readReadyBriefingSQL = `SELECT s.id,r.id,r.published_at,
 CASE WHEN octet_length(s.source_bytes)<=65536 THEN s.source_bytes ELSE NULL END,
 CASE WHEN octet_length(r.script_bytes)<=256000 THEN r.script_bytes ELSE NULL END,
 CASE WHEN octet_length(r.manifest_bytes)<=262144 THEN r.manifest_bytes ELSE NULL END,
 (SELECT count(*) FROM public.app_prosody_renditions all_r WHERE all_r.workspace_id=s.workspace_id AND all_r.item_id=s.id)
 FROM public.app_prosody_submissions s
 JOIN public.app_prosody_renditions r ON r.workspace_id=s.workspace_id AND r.item_id=s.id
 WHERE s.workspace_id=$1 AND s.feed_id=$2 AND s.id=$3 AND s.status='ready' AND s.deleted_at IS NULL
 ORDER BY r.published_at,r.id LIMIT 1 FOR SHARE OF s`
	readableFeedSQL = `SELECT id FROM public.app_prosody_feeds WHERE workspace_id=$1 AND id=$2 AND deleted_at IS NULL FOR SHARE`
	readActivitySQL = `SELECT s.id,s.status,s.accepted_at,
 CASE WHEN octet_length(s.source_bytes)<=65536 THEN s.source_bytes ELSE NULL END
 FROM public.app_prosody_submissions s
 WHERE s.workspace_id=$1 AND s.feed_id=$2 AND s.deleted_at IS NULL
 AND s.status IN ('processing','failed','ready') AND s.accepted_at<=$3
 AND ($4::boolean=false OR (s.accepted_at,s.id)>($5::timestamptz,$6::uuid))
 ORDER BY s.accepted_at,s.id LIMIT $7 FOR SHARE OF s`
)

// ReadArgument describes an existing source argument, not a native grant slot.
// Position is one-based. MaxUTF8Bytes applies only to string arguments; zero
// means a typed nonstring value, not an unbounded string allowance.
type ReadArgument struct {
	Position            int
	Name, SQLType, Role string
	MaxUTF8Bytes        int
	// Integer ceilings apply only to the pagination LIMIT argument. InitialMax
	// is a proposed future host limit; the existing source still permits SourceMax.
	SourceMax, InitialMax int
}

// ReadColumn describes the scanned source projection. MaxBytes applies to
// byte/string columns; zero denotes typed scalar encoding, not a wire budget.
// Bound explains whether SQL or source validation establishes the byte bound.
type ReadColumn struct {
	Name, SQLType string
	MaxBytes      int
	Bound         string
}

// ReadStatement is LOCAL immutable-source inventory, not native registration or
// semantic SQL admission. SHA256 covers exact UTF-8 SQL, including whitespace.
// SELECT ... FOR SHARE still requires a suitable caller-held transaction; phase
// "human-read" does not imply PostgreSQL READ ONLY mode.
type ReadStatement struct {
	ID, SQL                       string
	SHA256                        [32]byte
	Phase                         string
	WorkspaceParameter            int
	Arguments                     []ReadArgument
	Columns                       []ReadColumn
	SourceMaxRows, InitialMaxRows int
	// Variable payload only, excluding fixed scalar/transport encodings.
	MaxVariableBytesPerRow int
	NativeIdentitySlots    []string
	NativeIdentityNote     string
}

// ConsumptionReadCatalog returns independent mutable metadata on every call.
// Its six unique statements cover seven literal callsites. No callback,
// transaction, current-authority implementation or execution API is included.
func ConsumptionReadCatalog() []ReadStatement {
	uuid := func(position int, name, role string) ReadArgument {
		return ReadArgument{Position: position, Name: name, SQLType: "uuid", Role: role, MaxUTF8Bytes: 36}
	}
	timestamp := func(position int, name string) ReadArgument {
		return ReadArgument{Position: position, Name: name, SQLType: "timestamptz", Role: "cursor-location"}
	}
	limit := func(position int) ReadArgument {
		return ReadArgument{Position: position, Name: "lookahead_limit", SQLType: "integer", Role: "page-limit", SourceMax: 51, InitialMax: 11}
	}
	columns := func() []ReadColumn {
		return []ReadColumn{
			{Name: "item_id", SQLType: "uuid", MaxBytes: 36, Bound: "typed UUID"},
			{Name: "rendition_id", SQLType: "uuid", MaxBytes: 36, Bound: "typed UUID"},
			{Name: "published_at", SQLType: "timestamptz", Bound: "typed time.Time; nonzero source validation"},
			{Name: "source_bytes", SQLType: "bytea", MaxBytes: 65536, Bound: "SQL octet_length CASE; content codec rejects NULL/invalid"},
			{Name: "script_bytes", SQLType: "bytea", MaxBytes: 256000, Bound: "SQL octet_length CASE; source rejects NULL/invalid UTF-8"},
			{Name: "manifest_bytes", SQLType: "bytea", MaxBytes: 262144, Bound: "SQL octet_length CASE; source rejects NULL/invalid manifest"},
			{Name: "rendition_count", SQLType: "bigint", Bound: "source requires exactly 1"},
		}
	}
	statements := []ReadStatement{
		{ID: "feeds", SQL: readFeedsSQL, WorkspaceParameter: 1, Arguments: []ReadArgument{uuid(1, "workspace", "workspace"), uuid(2, "after_feed_id", "cursor-location"), limit(3)}, Columns: []ReadColumn{
			{Name: "feed_id", SQLType: "uuid", MaxBytes: 36, Bound: "typed UUID; source FeedID validation"},
			{Name: "kind", SQLType: "text", MaxBytes: 8, Bound: "source permits personal/team only"},
			{Name: "title", SQLType: "bytea", MaxBytes: 4096, Bound: "SQL octet_length CASE; source rejects empty/invalid UTF-8"},
		}, SourceMaxRows: 51, InitialMaxRows: 11, MaxVariableBytesPerRow: 4096},
		{ID: "clock", SQL: readClockSQL, WorkspaceParameter: 0, Columns: []ReadColumn{{Name: "now", SQLType: "timestamptz", Bound: "typed time.Time"}}, SourceMaxRows: 1, InitialMaxRows: 1},
		{ID: "readable-feed", SQL: readableFeedSQL, WorkspaceParameter: 1, Arguments: []ReadArgument{uuid(1, "workspace", "workspace"), uuid(2, "feed", "resource-feed")}, Columns: []ReadColumn{{Name: "feed_id", SQLType: "uuid", MaxBytes: 36, Bound: "typed UUID"}}, SourceMaxRows: 1, InitialMaxRows: 1},
		{ID: "activity", SQL: readActivitySQL, WorkspaceParameter: 1, Arguments: []ReadArgument{
			uuid(1, "workspace", "workspace"), uuid(2, "feed", "resource-feed"), timestamp(3, "cutoff"), {Position: 4, Name: "has_cursor", SQLType: "boolean", Role: "cursor-location"}, timestamp(5, "accepted_at"), uuid(6, "after_item_id", "cursor-location"), limit(7),
		}, Columns: []ReadColumn{
			{Name: "item_id", SQLType: "uuid", MaxBytes: 36, Bound: "typed UUID; source validation"},
			{Name: "status", SQLType: "text", MaxBytes: 10, Bound: "SQL enum predicate and source validation"},
			{Name: "accepted_at", SQLType: "timestamptz", Bound: "typed time.Time; source requires nonzero"},
			{Name: "source_bytes", SQLType: "bytea", MaxBytes: 65536, Bound: "SQL octet_length CASE; content codec rejects NULL/invalid"},
		}, SourceMaxRows: 51, InitialMaxRows: 11, MaxVariableBytesPerRow: 65536},
		{ID: "ready-list", SQL: readReadyBriefingsSQL, WorkspaceParameter: 1, Arguments: []ReadArgument{
			uuid(1, "workspace", "workspace"), uuid(2, "feed", "resource-feed"), timestamp(3, "cutoff"), {Position: 4, Name: "has_cursor", SQLType: "boolean", Role: "cursor-location"}, timestamp(5, "published_at"), uuid(6, "after_item_id", "cursor-location"), uuid(7, "after_rendition_id", "cursor-location"), limit(8),
		}, Columns: columns(), SourceMaxRows: 51, InitialMaxRows: 11, MaxVariableBytesPerRow: 583680},
		{ID: "ready-detail", SQL: readReadyBriefingSQL, WorkspaceParameter: 1, Arguments: []ReadArgument{uuid(1, "workspace", "workspace"), uuid(2, "feed", "resource-feed"), uuid(3, "item", "resource-item")}, Columns: columns(), SourceMaxRows: 1, InitialMaxRows: 1, MaxVariableBytesPerRow: 583680},
	}
	for i := range statements {
		statements[i].SHA256 = sha256.Sum256([]byte(statements[i].SQL))
		statements[i].Phase = "human-read"
		statements[i].NativeIdentitySlots = []string{}
		statements[i].NativeIdentityNote = "none: native person/session/grant/fence authority remains host-owned; workspace/resource arguments are descriptive bindings only"
	}
	return statements
}

// ReadRouteBudget is the proposed initial route's application-query budget.
// It excludes native authority/control queries and does not mount a route.
type ReadRouteBudget struct {
	ID                                                    string
	Statements                                            []string
	MaxCalls, MaxCallsPerStatement                        int
	SourcePageCeiling, InitialPageCeiling                 int
	InitialMaxRawPayloadBytes, InitialMaxEncodedJSONBytes int
}

// ConsumptionReadBudgets returns fresh metadata. Each named statement executes
// at most once per route call, including the shared clock and feed lock query.
// The initial 10/11 caps require future host enforcement; source remains 50/51.
func ConsumptionReadBudgets() []ReadRouteBudget {
	budgets := []ReadRouteBudget{
		{ID: "feeds", Statements: []string{"feeds"}, MaxCalls: 1, SourcePageCeiling: 50, InitialPageCeiling: 10},
		{ID: "activity", Statements: []string{"readable-feed", "clock", "activity"}, MaxCalls: 3, SourcePageCeiling: 50, InitialPageCeiling: 10},
		{ID: "ready-list", Statements: []string{"readable-feed", "clock", "ready-list"}, MaxCalls: 3, SourcePageCeiling: 50, InitialPageCeiling: 10},
		{ID: "ready-detail", Statements: []string{"readable-feed", "ready-detail"}, MaxCalls: 2, SourcePageCeiling: 1, InitialPageCeiling: 1},
	}
	for i := range budgets {
		budgets[i].MaxCallsPerStatement = 1
		budgets[i].InitialMaxRawPayloadBytes = 8 << 20
		budgets[i].InitialMaxEncodedJSONBytes = 1 << 20
	}
	return budgets
}
