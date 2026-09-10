import {
    CategoryScale,
    Chart as ChartJS,
    Filler,
    LinearScale,
    LineElement,
    PointElement,
    Tooltip,
} from 'chart.js';
import React, {useMemo} from 'react';
import {Line} from 'react-chartjs-2';
import {useSelector} from 'react-redux';

import {getTheme} from '../../redux/mmSelectors';
import type {ChannelPostCountByDuration, TimeRange, TopChannel} from '../../types';

ChartJS.register(CategoryScale, LinearScale, PointElement, LineElement, Tooltip, Filler);

// Adapted from the deprecated Mattermost
// `webapp/channels/src/components/activity_and_insights/insights/top_channels/top_channels_line_chart/top_channels_line_chart.tsx`
// (commit 26617fcbdc). Same five-line shape, same per-rank color palette
// from the user's theme (`buttonBg`, `onlineIndicator`, `awayIndicator`,
// `dndIndicator`, `newMessageSeparator`) so the lines tie back to the
// numbered list's leading dots. Uses `react-chartjs-2` directly rather
// than the original `components/analytics/line_chart` wrapper from the
// host webapp.

interface Props {
    topChannels: TopChannel[];
    postCountByDuration: ChannelPostCountByDuration;
    timeRange: TimeRange;
}

const FALLBACK_PALETTE = ['#1c58d9', '#3db887', '#ffbc1f', '#d24b4e', '#cc8f00'];

function paletteFor(theme: ReturnType<typeof getTheme>): string[] {
    if (!theme) {
        return FALLBACK_PALETTE;
    }
    return [
        theme.buttonBg ?? FALLBACK_PALETTE[0],
        theme.onlineIndicator ?? FALLBACK_PALETTE[1],
        theme.awayIndicator ?? FALLBACK_PALETTE[2],
        theme.dndIndicator ?? FALLBACK_PALETTE[3],
        theme.newMessageSeparator ?? FALLBACK_PALETTE[4],
    ];
}

// Buckets are always days now. Hour buckets existed only for the 'today'
// range, which went away with the daily snapshot.
function formatBucketLabel(bucket: string): string {
    // Day bucket — "YYYY-MM-DD". Render as "MMM DD".
    const d = new Date(bucket + 'T00:00:00');
    if (Number.isNaN(d.getTime())) {
        return bucket;
    }
    return d.toLocaleDateString([], {month: 'short', day: '2-digit'});
}

export const TopChannelsLineChart: React.FC<Props> = ({topChannels, postCountByDuration, timeRange}) => {
    const theme = useSelector(getTheme);
    const palette = paletteFor(theme);

    const {labels, rawLabels, channelSeries} = useMemo(() => {
        const sortedKeys = Object.keys(postCountByDuration).sort();
        const series: Record<string, number[]> = {};
        for (const channel of topChannels.slice(0, 5)) {
            series[channel.id] = sortedKeys.map((k) => postCountByDuration[k]?.[channel.id] ?? 0);
        }
        return {
            labels: sortedKeys.map((k) => formatBucketLabel(k)),
            rawLabels: sortedKeys,
            channelSeries: series,
        };
    }, [postCountByDuration, topChannels, timeRange]);

    const data = useMemo(() => {
        const datasets = topChannels.slice(0, 5).map((channel, index) => {
            const color = palette[index] ?? palette[0];
            return {
                label: channel.display_name || channel.name,
                data: channelSeries[channel.id] ?? [],
                borderColor: color,
                backgroundColor: 'transparent',
                pointBackgroundColor: color,
                pointBorderColor: 'transparent',
                pointRadius: 0,
                hoverBackgroundColor: color,
                hitRadius: 10,
                tension: 0.4,
            };
        });
        return {labels, datasets};
    }, [labels, channelSeries, topChannels, palette]);

    const tickColor = 'rgba(var(--center-channel-color-rgb), 0.72)';
    const options = useMemo(() => ({
        responsive: true as const,
        maintainAspectRatio: false as const,
        scales: {
            x: {
                grid: {drawOnChartArea: false},
                ticks: {
                    callback(_value: string | number, index: number) {
                        const label = labels[index] ?? '';

                        // 28 day buckets is too many to label individually;
                        // 7 fits. The old hour-bucket thinning went away with
                        // the 'today' range.
                        if (timeRange === '28_day') {
                            return index % 4 === 0 ? label : '';
                        }
                        return label;
                    },
                    font: {family: 'Open Sans', size: 10},
                    color: tickColor,
                },
            },
            y: {
                grid: {drawOnChartArea: true},
                beginAtZero: true,
                ticks: {
                    maxTicksLimit: 5,
                    precision: 0,
                    font: {family: 'Open Sans', size: 10},
                    color: tickColor,
                },
            },
        },
        plugins: {
            legend: {display: false as const},
            tooltip: {
                callbacks: {
                    label(context: {dataset: {label?: string}}) {
                        const label = context.dataset.label ?? '';
                        return label.length > 16 ? ` ${label.slice(0, 16)}…` : ` ${label}`;
                    },
                    title() {
                        return '';
                    },
                    footer(items: Array<{parsed: {y: number | null}}>) {
                        return `${items[0].parsed.y ?? 0} messages`;
                    },
                },
                bodyAlign: 'left' as const,
                bodySpacing: 10,
                footerAlign: 'center' as const,
                footerSpacing: 10,
                multiKeyBackground: 'transparent',
            },
        },
    }), [labels, timeRange, tickColor]);

    if (rawLabels.length === 0 || topChannels.length === 0) {
        return null;
    }

    return (
        <div className='top-channels-line-chart'>
            <Line
                data={data}
                options={options}
            />
        </div>
    );
};
