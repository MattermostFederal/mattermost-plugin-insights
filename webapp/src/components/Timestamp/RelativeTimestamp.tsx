// RelativeTimestamp wraps the host's `<Timestamp/>` (exposed at runtime
// via `window.Components.Timestamp` per
// webapp/channels/src/plugins/export.ts) when present, and falls back to
// `Intl.RelativeTimeFormat` otherwise. The deprecated insights rows used
// the host component with custom `units` lists like:
//
//   <Timestamp value={ms} units={['now','minute','hour','day','week','month']} useTime={false}/>
//
// We accept the same `units` prop and forward it to the host component if
// available; the fallback ignores it (it picks the largest unit that
// fits) but renders an equivalent "3 days ago" string.

import React, {memo} from 'react';

type TimestampUnit = 'now' | 'minute' | 'hour' | 'day' | 'week' | 'month' | 'year';

interface HostTimestampProps {
    value: number;
    units?: TimestampUnit[];
    useTime?: boolean;
    style?: 'long' | 'short' | 'numeric';
    day?: 'numeric' | '2-digit';
}

interface MMComponents {
    Timestamp?: React.ComponentType<HostTimestampProps>;
}

interface Props extends HostTimestampProps {
    className?: string;
}

const FALLBACK_UNITS: Array<{limitMs: number; divisor: number; unit: Intl.RelativeTimeFormatUnit}> = [
    {limitMs: 60_000, divisor: 1_000, unit: 'second'},
    {limitMs: 3_600_000, divisor: 60_000, unit: 'minute'},
    {limitMs: 86_400_000, divisor: 3_600_000, unit: 'hour'},
    {limitMs: 7 * 86_400_000, divisor: 86_400_000, unit: 'day'},
    {limitMs: 30 * 86_400_000, divisor: 7 * 86_400_000, unit: 'week'},
    {limitMs: 365 * 86_400_000, divisor: 30 * 86_400_000, unit: 'month'},
];

function fallbackFormat(unixMillis: number): string {
    if (!unixMillis) {
        return '';
    }
    const fmt = new Intl.RelativeTimeFormat(undefined, {numeric: 'auto'});
    const deltaMs = unixMillis - Date.now();
    const absMs = Math.abs(deltaMs);
    for (const u of FALLBACK_UNITS) {
        if (absMs < u.limitMs) {
            return fmt.format(Math.round(deltaMs / u.divisor), u.unit);
        }
    }
    return fmt.format(Math.round(deltaMs / (365 * 86_400_000)), 'year');
}

const RelativeTimestampComponent: React.FC<Props> = ({className, ...props}) => {
    const HostTimestamp = ((typeof window === 'undefined' ? undefined : (window as unknown as {Components?: MMComponents}).Components?.Timestamp));
    if (HostTimestamp) {
        return (
            <span className={className}>
                <HostTimestamp {...props}/>
            </span>
        );
    }
    return <span className={className}>{fallbackFormat(props.value)}</span>;
};

export const RelativeTimestamp = memo(RelativeTimestampComponent);
