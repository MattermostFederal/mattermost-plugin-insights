import React, {useEffect} from 'react';
import {useSelector} from 'react-redux';

import {getCurrentTeamId, getCurrentUserId} from '../redux/mmSelectors';
import {trackInsightsEvent} from '../utils/telemetry';

import {InsightsPage} from './Page/InsightsPage';

// Mirrors the deprecated `LocalStorageStore.setPenultimate/PreviousViewedType`
// behavior in `insights.tsx`. The host webapp persists which "page type"
// the user was last viewing (`channel`, `threads`, `insights`, etc) so the
// browser-back / "Switch back to channel" affordance can route correctly.
// Key shape mirrors the host store: `previous_viewed_type:<userID>:<teamID>`.
function persistPreviousViewedType(userID: string, teamID: string) {
    if (!userID || !teamID || typeof window === 'undefined') {
        return;
    }
    const previousKey = `previous_viewed_type:${userID}:${teamID}`;
    const penultimateKey = `penultimate_viewed_type:${userID}:${teamID}`;
    const previous = window.localStorage.getItem(previousKey);
    if (previous && previous !== 'insights') {
        window.localStorage.setItem(penultimateKey, previous);
    }
    window.localStorage.setItem(previousKey, 'insights');
}

export const InsightsProduct: React.FC = () => {
    const userID = useSelector(getCurrentUserId);
    const teamID = useSelector(getCurrentTeamId);

    useEffect(() => {
        // Mirrors the deprecated `sidebar_open_insights` track event,
        // fired the first time the user lands on the Insights product.
        trackInsightsEvent('sidebar_open_insights');
        persistPreviousViewedType(userID, teamID);
        document.body.classList.add('insights-product-active');
        return () => {
            document.body.classList.remove('insights-product-active');
        };
    }, [userID, teamID]);
    return <InsightsPage/>;
};
