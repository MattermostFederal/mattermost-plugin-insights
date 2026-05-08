// Ported verbatim from mattermost/mattermost
// webapp/platform/components/src/skeleton_loader/index.tsx (commit 26617fcbdc).

import styled, {keyframes} from 'styled-components';

const skeletonFade = keyframes`
    0% {
        background-color: rgba(var(--center-channel-color-rgb), 0.08);
    }
    50% {
        background-color: rgba(var(--center-channel-color-rgb), 0.16);
    }
    100% {
        background-color: rgba(var(--center-channel-color-rgb), 0.08);
    }
`;

const BaseLoader = styled.div`
    animation-duration: 1500ms;
    animation-iteration-count: infinite;
    animation-name: ${skeletonFade};
    animation-timing-function: ease-in-out;
    background-color: rgba(var(--center-channel-color-rgb), 0.08);
`;

export interface CircleSkeletonLoaderProps {
    size: string | number;
}

export const CircleSkeletonLoader = styled(BaseLoader)<CircleSkeletonLoaderProps>`
    display: block;
    border-radius: 50%;
    height: ${(props) => getCorrectSizeDimension(props.size)};
    width: ${(props) => getCorrectSizeDimension(props.size)};
`;

export interface RectangleSkeletonLoaderProps {
    height: string | number;
    width?: string | number;
    borderRadius?: number;
    margin?: string;
    flex?: string;
}

export const RectangleSkeletonLoader = styled(BaseLoader)<RectangleSkeletonLoaderProps>`
    height: ${(props) => getCorrectSizeDimension(props.height)};
    width: ${(props) => getCorrectSizeDimension(props.width, '100%')};
    border-radius: ${(props) => props?.borderRadius ?? 8}px;
    margin: ${(props) => props?.margin ?? null};
    flex: ${(props) => props?.flex ?? null};
`;

function getCorrectSizeDimension(size: number | string | undefined, fallback: string | null = null) {
    if (size) {
        return (typeof size === 'string') ? size : `${size}px`;
    }

    return fallback;
}
