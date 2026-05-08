// Ported from mattermost/mattermost
// webapp/channels/src/components/activity_and_insights/insights/top_threads/top_threads_item/top_threads_item.tsx
// (commit 26617fcbdc^).
//
// Adaptations:
//   - `<Avatar url={imageURLForUser(...)}/>` → plugin `<HostAvatar/>` which
//     uses the host's `Avatar` and `imageURLForUser` when available
//     (registered at `window.Components`), falls back to a plain <img>.
//   - `<Tag text={channel.display_name}/>` → plugin `<Tag/>` (port).
//   - `<Markdown message={post.message}>` → host's
//     `window.PostUtils.formatText` + `messageHtmlToComponent` when
//     present; falls back to a plain-text excerpt.
//   - `dispatch(selectPostAndParentChannel)` → `openThreadRHS` helper
//     (handles host's Redux thunk dispatch correctly).
//   - Compliance-export "join channel" branch: when the user is NOT a
//     member of the thread's channel AND the server has compliance export
//     enabled (license.Compliance && config.EnableComplianceExport), the
//     deprecated row click opened a `<JoinChannelModal/>` to confirm.
//     Same flow ported here. Otherwise, click opens the RHS as usual.

import React, {memo, useCallback, useState} from 'react';
import {FormattedMessage} from 'react-intl';
import {useDispatch, useSelector} from 'react-redux';

import {getConfig, getCurrentUserId, getLicense, isMemberOfChannel} from '../../redux/mmSelectors';
import type {Status} from '../../redux/types';
import type {TopThread} from '../../types';
import {navigateTo, openThreadRHS} from '../../utils/navigation';
import {trackInsightsEvent} from '../../utils/telemetry';
import {HostAvatar} from '../Avatar/HostAvatar';
import {WidgetEmptyState} from '../EmptyState/WidgetEmptyState';
import {JoinChannelModal} from '../Modal/JoinChannelModal';
import {TopThreadsSkeleton} from '../Skeleton/CardSkeletons';
import {Tag} from '../Tag/Tag';

interface Props {
    items: TopThread[];
    hasNext: boolean;
    status: Status;
    error?: string;
    teamName?: string;
}

function authorName(thread: TopThread): string {
    const u = thread.user_information;
    const fl = `${u.first_name} ${u.last_name}`.trim();
    return fl || u.username;
}

function replyCount(thread: TopThread): number {
    const rc = (thread.post as {reply_count?: number} | undefined)?.reply_count;
    return typeof rc === 'number' ? rc : 0;
}

interface PostUtils {
    formatText?: (text: string, options?: Record<string, unknown>) => string;
    messageHtmlToComponent?: (html: string, options?: Record<string, unknown>) => React.ReactNode;
}

function hostPostUtils(): PostUtils | undefined {
    if (typeof window === 'undefined') {
        return undefined;
    }
    return (window as unknown as {PostUtils?: PostUtils}).PostUtils;
}

const PostPreview: React.FC<{post: TopThread['post']}> = ({post}) => {
    const message = (post as {message?: string})?.message ?? '';
    if (!message) {
        return <span className='preview-fallback'/>;
    }
    const utils = hostPostUtils();
    if (utils?.formatText && utils?.messageHtmlToComponent) {
        const html = utils.formatText(message, {singleline: true, mentionHighlight: false, atMentions: false});
        return <>{utils.messageHtmlToComponent(html, {})}</>;
    }
    return <>{message}</>;
};

const ThreadRow: React.FC<{thread: TopThread; teamName?: string}> = ({thread, teamName}) => {
    const dispatch = useDispatch();
    const currentUserId = useSelector(getCurrentUserId);
    const isChannelMember = useSelector((state: unknown) => isMemberOfChannel(state, thread.channel_id));
    const license = useSelector(getLicense);
    const config = useSelector(getConfig);
    const [joinModalOpen, setJoinModalOpen] = useState(false);

    const complianceExportEnabled =
        license.Compliance === 'true' && config.EnableComplianceExport === 'true';
    const requiresJoin = !isChannelMember && complianceExportEnabled;

    const openRHS = useCallback(() => {
        const postId = thread.post?.id;
        if (!postId) {
            return;
        }
        const opened = openThreadRHS(postId, dispatch as unknown as (a: unknown) => unknown);
        if (opened) {
            return;
        }
        if (teamName) {
            navigateTo(`/${teamName}/pl/${postId}`);
        }
    }, [dispatch, teamName, thread.post]);

    const handleClick = useCallback(() => {
        trackInsightsEvent('open_thread_from_top_threads_widget');
        if (requiresJoin) {
            setJoinModalOpen(true);
            return;
        }
        openRHS();
    }, [requiresJoin, openRHS]);

    return (
        <>
            <div
                className='thread-item'
                onClick={handleClick}
                role='button'
                tabIndex={0}
            >
                <div className='thread-details'>
                    <HostAvatar
                        userID={thread.user_information.id}
                        lastPictureUpdate={thread.user_information.last_picture_update}
                        size='xs'
                    />
                    <span className='display-name'>{authorName(thread)}</span>
                    <Tag text={thread.channel_display_name || thread.channel_name}/>
                    <div className='reply-count'>
                        <i className='icon icon-reply-outline'/>
                        <span>{replyCount(thread)}</span>
                    </div>
                </div>
                <div className='preview'>
                    {requiresJoin ? (
                        <span className='compliance-information'>
                            <FormattedMessage
                                id='insights.topThreadItem.notChannelMember'
                                defaultMessage="You'll need to join the {channel} channel to see this thread."
                                values={{
                                    channel: <strong>{thread.channel_display_name || thread.channel_name}</strong>,
                                }}
                            />
                        </span>
                    ) : (
                        <PostPreview post={thread.post}/>
                    )}
                </div>
            </div>
            {joinModalOpen ? (
                <JoinChannelModal
                    show={true}
                    thread={thread}
                    currentUserId={currentUserId}
                    onJoined={() => openRHS()}
                    onExited={() => setJoinModalOpen(false)}
                />
            ) : null}
        </>
    );
};

const TopThreadsListComponent: React.FC<Props> = ({items, status, error, teamName}) => {
    if (status === 'loading') {
        return <TopThreadsSkeleton/>;
    }
    if (status === 'error') {
        return (
            <p className='insights-card__error'>{error || (
                <FormattedMessage
                    id='insights.topThreads.failed'
                    defaultMessage='Failed to load threads'
                />
            )}</p>
        );
    }
    if (items.length === 0) {
        return <WidgetEmptyState icon='message-text-outline'/>;
    }

    return (
        <div className='thread-list'>
            {items.map((item, idx) => (
                <ThreadRow
                    key={`${item.channel_id}-${idx}`}
                    thread={item}
                    teamName={teamName}
                />
            ))}
        </div>
    );
};

export const TopThreadsList = memo(TopThreadsListComponent);
