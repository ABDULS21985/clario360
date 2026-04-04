package profiler

import (
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/clario360/platform/internal/cyber/ueba/model"
)

const (
	databaseCapacity = 20
	tableCapacity    = 20
	sourceIPCapacity = 50
)

type BehavioralProfiler struct {
	alpha float64
}

func NewBehavioralProfiler(alpha float64) *BehavioralProfiler {
	return &BehavioralProfiler{alpha: alpha}
}

func (p *BehavioralProfiler) UpdateProfile(profile *model.UEBAProfile, event *model.DataAccessEvent) error {
	if profile == nil || event == nil {
		return fmt.Errorf("profile and event are required")
	}
	profile.EnsureDefaults()
	if profile.FirstSeenAt.IsZero() {
		profile.FirstSeenAt = event.EventTimestamp.UTC()
	}

	p.finalizePriorDay(profile, event.EventTimestamp.UTC())
	p.updateAccessTimes(profile, event.EventTimestamp.UTC())
	p.updateDataVolume(profile, event)
	p.updateAccessPatterns(profile, event)
	p.updateSourceIPs(profile, event)
	p.updateSessionStats(profile, event)
	p.updateFailureStats(profile, event)
	p.updateObservationState(profile, event.EventTimestamp.UTC())

	return nil
}

func (p *BehavioralProfiler) updateAccessTimes(profile *model.UEBAProfile, ts time.Time) {
	hour := ts.Hour()
	weekday := remapWeekday(ts.Weekday())
	profile.Baseline.AccessTimes.HourlyDistribution = UpdateOneHotDistribution24(profile.Baseline.AccessTimes.HourlyDistribution, hour, p.alpha)
	profile.Baseline.AccessTimes.DailyDistribution = UpdateOneHotDistribution7(profile.Baseline.AccessTimes.DailyDistribution, weekday, p.alpha)
	profile.Baseline.AccessTimes.PeakHours = PeakHours(profile.Baseline.AccessTimes.HourlyDistribution, 3)
	profile.Baseline.AccessTimes.ActiveHoursCount = ActiveHoursCount(profile.Baseline.AccessTimes.HourlyDistribution, 0.01)
}

func (p *BehavioralProfiler) updateDataVolume(profile *model.UEBAProfile, event *model.DataAccessEvent) {
	state := &profile.Baseline.State
	state.CurrentDayBytes += float64(event.BytesAccessed)
	state.CurrentDayRows += float64(event.RowsAccessed)

	if float64(event.BytesAccessed) > profile.Baseline.DataVolume.MaxSingleQueryBytes {
		profile.Baseline.DataVolume.MaxSingleQueryBytes = float64(event.BytesAccessed)
	}
	if float64(event.RowsAccessed) > profile.Baseline.DataVolume.MaxSingleQueryRows {
		profile.Baseline.DataVolume.MaxSingleQueryRows = float64(event.RowsAccessed)
	}
}

func (p *BehavioralProfiler) updateAccessPatterns(profile *model.UEBAProfile, event *model.DataAccessEvent) {
	if db := strings.TrimSpace(event.DatabaseName); db != "" {
		profile.Baseline.AccessPatterns.DatabasesAccessed = upsertLRUString(profile.Baseline.AccessPatterns.DatabasesAccessed, db, databaseCapacity)
	}
	if table := qualifiedTableName(event.SchemaName, event.TableName); table != "" {
		profile.Baseline.AccessPatterns.TablesAccessed = updateTableFrequencies(profile.Baseline.AccessPatterns.TablesAccessed, table, event.EventTimestamp.UTC(), p.alpha, tableCapacity)
	}

	queryType := normalizeQueryType(event.Action)
	profile.Baseline.AccessPatterns.QueryTypes = updateQueryTypeDistribution(profile.Baseline.AccessPatterns.QueryTypes, queryType, p.alpha)

	if event.DurationMS > 0 {
		baseline := &profile.Baseline.AccessPatterns
		mean, m2, stddev := WelfordUpdate(profile.ObservationCount, baseline.AvgQueryDurationMS, baseline.AvgQueryDurationM2, float64(event.DurationMS))
		baseline.AvgQueryDurationMS = mean
		baseline.AvgQueryDurationM2 = m2
		baseline.AvgQueryDurationStddev = stddev
	}
}

func (p *BehavioralProfiler) updateSourceIPs(profile *model.UEBAProfile, event *model.DataAccessEvent) {
	if ip := strings.TrimSpace(event.SourceIP); ip != "" {
		profile.Baseline.SourceIPs = upsertLRUString(profile.Baseline.SourceIPs, ip, sourceIPCapacity)
	}
}

func (p *BehavioralProfiler) updateSessionStats(profile *model.UEBAProfile, event *model.DataAccessEvent) {
	state := &profile.Baseline.State
	switch event.Action {
	case "login":
		state.CurrentDaySessions++
		state.CurrentSessionStartUnix = event.EventTimestamp.UTC().Unix()
	case "logout":
		if state.CurrentSessionStartUnix > 0 {
			durationMinutes := event.EventTimestamp.UTC().Sub(time.Unix(state.CurrentSessionStartUnix, 0)).Minutes()
			if durationMinutes > 0 {
				baseline := &profile.Baseline.SessionStats
				mean, m2, stddev := WelfordUpdate(profile.ObservationCount, baseline.AvgSessionDurationMinutes, baseline.AvgSessionDurationM2, durationMinutes)
				baseline.AvgSessionDurationMinutes = mean
				baseline.AvgSessionDurationM2 = m2
				baseline.AvgSessionDurationStddev = stddev
			}
			state.CurrentSessionStartUnix = 0
		}
	}
}

func (p *BehavioralProfiler) updateFailureStats(profile *model.UEBAProfile, event *model.DataAccessEvent) {
	state := &profile.Baseline.State
	if event.Action != "logout" {
		state.CurrentDayQueries++
	}
	if !event.Success {
		state.CurrentDayFailures++
	}
	if state.CurrentDayQueries > 0 {
		profile.Baseline.FailureRate.FailureRatePercent = (state.CurrentDayFailures / state.CurrentDayQueries) * 100
	}
}

func (p *BehavioralProfiler) updateObservationState(profile *model.UEBAProfile, ts time.Time) {
	day := ts.Format("2006-01-02")
	if day != profile.Baseline.State.LastActiveDay {
		profile.DaysActive++
		profile.Baseline.State.LastActiveDay = day
	}
	profile.ObservationCount++
	profile.LastSeenAt = ts
	profile.ProfileMaturity = ClassifyMaturity(profile.ObservationCount, profile.DaysActive)
}

func (p *BehavioralProfiler) finalizePriorDay(profile *model.UEBAProfile, eventTime time.Time) {
	state := &profile.Baseline.State
	currentDay := eventTime.Format("2006-01-02")
	if state.CurrentDay == "" {
		state.CurrentDay = currentDay
		return
	}
	if state.CurrentDay == currentDay {
		return
	}

	dv := &profile.Baseline.DataVolume
	bytesMean, bytesM2, bytesStddev := WelfordUpdate(int64(maxInt(profile.DaysActive-1, 0)), dv.DailyBytesMean, dv.DailyBytesM2, state.CurrentDayBytes)
	dv.DailyBytesMean = bytesMean
	dv.DailyBytesM2 = bytesM2
	dv.DailyBytesStddev = bytesStddev

	rowsMean, rowsM2, rowsStddev := WelfordUpdate(int64(maxInt(profile.DaysActive-1, 0)), dv.DailyRowsMean, dv.DailyRowsM2, state.CurrentDayRows)
	dv.DailyRowsMean = rowsMean
	dv.DailyRowsM2 = rowsM2
	dv.DailyRowsStddev = rowsStddev

	ss := &profile.Baseline.SessionStats
	sessionMean, sessionM2, sessionStddev := WelfordUpdate(int64(maxInt(profile.DaysActive-1, 0)), ss.DailySessionCountMean, ss.DailySessionCountM2, state.CurrentDaySessions)
	ss.DailySessionCountMean = sessionMean
	ss.DailySessionCountM2 = sessionM2
	ss.DailySessionCountStddev = sessionStddev

	fr := &profile.Baseline.FailureRate
	failureMean, failureM2, failureStddev := WelfordUpdate(int64(maxInt(profile.DaysActive-1, 0)), fr.DailyFailureCountMean, fr.DailyFailureCountM2, state.CurrentDayFailures)
	fr.DailyFailureCountMean = failureMean
	fr.DailyFailureCountM2 = failureM2
	fr.DailyFailureCountStddev = failureStddev
	if state.CurrentDayQueries > 0 {
		fr.FailureRatePercent = EMA(fr.FailureRatePercent, (state.CurrentDayFailures/state.CurrentDayQueries)*100, p.alpha)
	}

	state.CurrentDay = currentDay
	state.CurrentDayBytes = 0
	state.CurrentDayRows = 0
	state.CurrentDayFailures = 0
	state.CurrentDayQueries = 0
	state.CurrentDaySessions = 0
}

func remapWeekday(day time.Weekday) int {
	switch day {
	case time.Sunday:
		return 6
	default:
		return int(day - 1)
	}
}

func normalizeQueryType(action string) string {
	switch strings.ToLower(strings.TrimSpace(action)) {
	case "select", "insert", "update", "delete":
		return strings.ToLower(strings.TrimSpace(action))
	case "create", "alter", "drop":
		return "ddl"
	default:
		return "select"
	}
}

func updateQueryTypeDistribution(current map[string]float64, activeType string, alpha float64) map[string]float64 {
	if current == nil {
		current = map[string]float64{
			"select": 0,
			"insert": 0,
			"update": 0,
			"delete": 0,
			"ddl":    0,
		}
	}
	total := 0.0
	for _, key := range []string{"select", "insert", "update", "delete", "ddl"} {
		value := 0.0
		if key == activeType {
			value = 1
		}
		current[key] = EMA(current[key], value, alpha)
		total += current[key]
	}
	if total == 0 {
		return current
	}
	for key := range current {
		current[key] /= total
	}
	return current
}

func upsertLRUString(existing []string, value string, capacity int) []string {
	if value == "" {
		return existing
	}
	next := make([]string, 0, capacity)
	next = append(next, value)
	for _, item := range existing {
		if item == value {
			continue
		}
		next = append(next, item)
		if len(next) == capacity {
			break
		}
	}
	return next
}

func qualifiedTableName(schemaName, tableName string) string {
	schemaName = strings.TrimSpace(schemaName)
	tableName = strings.TrimSpace(tableName)
	switch {
	case schemaName == "" && tableName == "":
		return ""
	case schemaName == "":
		return tableName
	case tableName == "":
		return schemaName
	default:
		return schemaName + "." + tableName
	}
}

func updateTableFrequencies(existing []model.FrequencyEntry, table string, ts time.Time, alpha float64, capacity int) []model.FrequencyEntry {
	for i := range existing {
		if existing[i].Name == table {
			existing[i].Frequency = EMA(existing[i].Frequency, 1, alpha)
			existing[i].LastAccessed = ts
		} else {
			existing[i].Frequency = EMA(existing[i].Frequency, 0, alpha)
		}
	}

	found := false
	for i := range existing {
		if existing[i].Name == table {
			found = true
			break
		}
	}
	if !found {
		existing = append(existing, model.FrequencyEntry{
			Name:         table,
			Frequency:    alpha,
			LastAccessed: ts,
		})
	}

	sort.SliceStable(existing, func(i, j int) bool {
		if existing[i].Frequency == existing[j].Frequency {
			return existing[i].LastAccessed.After(existing[j].LastAccessed)
		}
		return existing[i].Frequency > existing[j].Frequency
	})
	if len(existing) > capacity {
		existing = existing[:capacity]
	}
	return existing
}

func maxInt(left, right int) int {
	if left > right {
		return left
	}
	return right
}
