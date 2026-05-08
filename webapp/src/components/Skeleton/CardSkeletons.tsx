// Per-insight skeleton compositions ported verbatim from the deprecated
// activity_and_insights cards (commit 26617fcbdc).

import React from 'react';

import {CircleSkeletonLoader, RectangleSkeletonLoader} from './SkeletonLoader';

export const TopReactionsSkeleton: React.FC = () => {
    const barChartHeights = [140, 178, 140, 120, 140];
    const entries: React.ReactNode[] = [];
    for (let i = 0; i < 5; i++) {
        entries.push(
            <div
                className='bar-chart-entry'
                key={i}
            >
                <RectangleSkeletonLoader
                    width={8}
                    height={barChartHeights[i]}
                    borderRadius={6}
                    margin='0 0 6px 0'
                />
                <CircleSkeletonLoader size={20}/>
            </div>,
        );
    }
    return <div className='top-reaction-skeleton'>{entries}</div>;
};

export const TopChannelsSkeleton: React.FC = () => {
    const skeletonFlexes = ['1', '0.85', '0.9', '0.5', '0.7'];
    const titles: React.ReactNode[] = [];
    for (let i = 0; i < 5; i++) {
        titles.push(
            <div
                className='top-channel-loading-row'
                key={i}
            >
                <CircleSkeletonLoader size={16}/>
                <RectangleSkeletonLoader
                    height={12}
                    margin='0 0 0 8px'
                    flex={skeletonFlexes[i]}
                />
            </div>,
        );
    }
    return (
        <div className='top-channel-skeleton'>
            <div className='top-channel-line-chart-skeleton'>
                <RectangleSkeletonLoader
                    height={200}
                    borderRadius={4}
                />
            </div>
            <div className='top-channel-list-skeleton'>{titles}</div>
        </div>
    );
};

export const TopThreadsSkeleton: React.FC = () => {
    const entries: React.ReactNode[] = [];
    for (let i = 0; i < 3; i++) {
        entries.push(
            <div
                className='top-thread-loading-container'
                key={i}
            >
                <div className='top-thread-loading-row'>
                    <CircleSkeletonLoader size={20}/>
                    <RectangleSkeletonLoader
                        height={12}
                        margin='0 0 0 8px'
                        flex='0.5'
                    />
                </div>
                <div>
                    <RectangleSkeletonLoader
                        height={8}
                        margin='0 0 8px 0'
                    />
                    <RectangleSkeletonLoader height={8}/>
                </div>
            </div>,
        );
    }
    return <div className='top-thread-skeleton'>{entries}</div>;
};

export const TopDMsSkeleton: React.FC = () => {
    const entries: React.ReactNode[] = [];
    for (let i = 0; i < 5; i++) {
        entries.push(
            <div
                className='dms-loading-container'
                key={i}
            >
                <CircleSkeletonLoader size={72}/>
                <div className='title-line'>
                    <RectangleSkeletonLoader
                        height={12}
                        flex='1'
                    />
                </div>
                <div>
                    <RectangleSkeletonLoader
                        height={8}
                        width={92}
                        margin='0 0 12px 0'
                    />
                    <RectangleSkeletonLoader
                        height={8}
                        width={72}
                    />
                </div>
            </div>,
        );
    }
    return <div className='top-dms-skeleton'>{entries}</div>;
};

export const TopInactiveChannelsSkeleton: React.FC = () => {
    const entries: React.ReactNode[] = [];
    for (let i = 0; i < 4; i++) {
        entries.push(
            <div
                className='least-active-channels-loading-container'
                key={i}
            >
                <CircleSkeletonLoader size={16}/>
                <RectangleSkeletonLoader
                    width='30%'
                    height={12}
                    margin='0 0 0 6px'
                    flex='1'
                />
                <RectangleSkeletonLoader
                    width='30%'
                    height={12}
                    margin='0 0 0 30px'
                    flex='1'
                />
                <RectangleSkeletonLoader
                    width='20%'
                    height={12}
                    margin='0 0 0 50px'
                    flex='1'
                />
            </div>,
        );
    }
    return <div className='least-active-channels-skeleton'>{entries}</div>;
};

export const TopBoardsSkeleton: React.FC = () => {
    const entries: React.ReactNode[] = [];
    for (let i = 0; i < 4; i++) {
        entries.push(
            <div
                className='top-board-loading-container'
                key={i}
            >
                <CircleSkeletonLoader size={32}/>
                <div className='loading-lines'>
                    <RectangleSkeletonLoader
                        height={12}
                        flex='none'
                    />
                    <RectangleSkeletonLoader
                        height={8}
                        flex='none'
                        margin='6px 0 0 0'
                    />
                </div>
            </div>,
        );
    }
    return <div className='top-board-skeleton'>{entries}</div>;
};

export const TopPlaybooksSkeleton: React.FC = () => {
    const entries: React.ReactNode[] = [];
    for (let i = 0; i < 3; i++) {
        entries.push(
            <div
                className='top-playbooks-loading-container'
                key={i}
            >
                <RectangleSkeletonLoader
                    height={12}
                    margin='0 0 8px 0'
                />
                <RectangleSkeletonLoader
                    height={8}
                    margin='0 0 8px 0'
                    width='80%'
                />
            </div>,
        );
    }
    return <div className='top-playbooks-skeleton'>{entries}</div>;
};

export const NewTeamMembersSkeleton: React.FC = () => {
    const entries: React.ReactNode[] = [];
    for (let i = 0; i < 5; i++) {
        entries.push(
            <div
                className='new-members-loading-container'
                key={i}
            >
                <CircleSkeletonLoader size={72}/>
                <div className='title-line'>
                    <RectangleSkeletonLoader
                        height={12}
                        flex='1'
                    />
                </div>
                <div>
                    <RectangleSkeletonLoader
                        height={8}
                        width={92}
                        margin='0 0 12px 0'
                    />
                    <RectangleSkeletonLoader
                        height={8}
                        width={72}
                    />
                </div>
            </div>,
        );
    }
    return <div className='new-members-skeleton'>{entries}</div>;
};
