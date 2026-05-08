// Ported from mattermost/mattermost
// webapp/channels/src/components/activity_and_insights/insights/top_reactions/top_reactions_bar_chart/top_reactions_bar_chart.tsx
// (commit 26617fcbdc).
//
// Adaptations:
//   - The deprecated source rendered emojis through the host webapp's
//     `RenderEmoji` component (Redux-bound + theme-aware). The plugin
//     can't import host components, so we render an `<img>` pointing at
//     the same `/static/emoji/<unified>.png` URL the host's
//     `getEmojiImageUrl` produces, with a server-name lookup fallback for
//     custom emojis. Fallback if the image fails: render `:name:` text.
//   - SimpleTooltip → native `title` attribute.

import React, {memo, useCallback} from 'react';

import type {TopReaction} from '../../types';
import {getEmojiImageUrl} from '../../utils/emoji';

interface Props {
    reactions: TopReaction[];
}

const EMOJI_SIZE = 20;
const MAX_BAR_HEIGHT = 156;

const ReactionEmoji: React.FC<{name: string}> = ({name}) => {
    const url = getEmojiImageUrl(name);
    const [errored, setErrored] = React.useState(false);
    if (!url || errored) {
        return <span className='top-reactions__bar-emoji-fallback'>{`:${name}:`}</span>;
    }
    return (
        <img
            src={url}
            alt={`:${name}:`}
            width={EMOJI_SIZE}
            height={EMOJI_SIZE}
            className='emoticon'
            onError={() => setErrored(true)}
        />
    );
};

const TopReactionsBarChartComponent: React.FC<Props> = ({reactions}) => {
    const renderEntries = useCallback(() => {
        if (!reactions.length) {
            return null;
        }
        const highestCount = reactions[0].count;
        return reactions.map((reaction) => {
            let barHeight = (reaction.count / highestCount) * MAX_BAR_HEIGHT;
            if (highestCount === reaction.count) {
                barHeight = MAX_BAR_HEIGHT;
            }
            return (
                <div
                    className='bar-chart-entry'
                    key={reaction.emoji_name}
                >
                    <span className='reaction-count'>{reaction.count}</span>
                    <div
                        className='bar-chart-data'
                        style={{height: `${barHeight}px`}}
                    />
                    <span title={reaction.emoji_name}>
                        <ReactionEmoji name={reaction.emoji_name}/>
                    </span>
                </div>
            );
        });
    }, [reactions]);

    return <div className='top-reaction-container'>{renderEntries()}</div>;
};

export const TopReactionsBarChart = memo(TopReactionsBarChartComponent);
