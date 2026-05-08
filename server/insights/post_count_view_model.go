package insights

import "time"

// ToChannelPostCountByDuration assembles the response-shaped chart map from
// the raw store rows. It walks every duration bucket from `start` to `now`
// in the appropriate step (hour or day) and pre-fills each channel's count
// with zero so empty buckets are visible to the client.
//
// numDays controls the bucket size: 1 -> hour buckets keyed by RFC3339; >1
// -> day buckets keyed by "YYYY-MM-DD". Direct port of
// model.ToDailyPostCountViewModel from the deleted server/public/model/insights.go.
func ToChannelPostCountByDuration(rows []*DurationPostCount, start *time.Time, numDays int, channelIDs []string) ChannelPostCountByDuration {
	out := ChannelPostCountByDuration{}
	keyTime := *start
	now := time.Now().In(start.Location())

	if numDays == 1 {
		for keyTime.Before(now) {
			out[keyTime.Format(time.RFC3339)] = blankChannelCounts(channelIDs)
			keyTime = keyTime.Add(time.Hour)
		}
	} else {
		for keyTime.Before(now) {
			out[keyTime.Format("2006-01-02")] = blankChannelCounts(channelIDs)
			keyTime = keyTime.Add(24 * time.Hour)
		}
	}

	for _, row := range rows {
		var parseFmt, keyFmt string
		if numDays == 1 {
			parseFmt = "2006-01-02T15"
			keyFmt = time.RFC3339
		} else {
			parseFmt = "2006-01-02"
			keyFmt = parseFmt
		}
		t, err := time.ParseInLocation(parseFmt, row.Duration, start.Location())
		if err != nil {
			continue
		}
		key := t.Format(keyFmt)
		bucket, ok := out[key]
		if !ok {
			bucket = map[string]int{}
			out[key] = bucket
		}
		bucket[row.ChannelID] = row.PostCount
	}

	return out
}

func blankChannelCounts(channelIDs []string) map[string]int {
	m := make(map[string]int, len(channelIDs))
	for _, id := range channelIDs {
		m[id] = 0
	}
	return m
}
