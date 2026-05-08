// The product's "Insights" label is rendered by Mattermost itself from the
// `switcherText` we pass to `registry.registerProduct(...)`, placed at the
// left of the global header next to the switcherIcon (the same way Boards
// and Playbooks render their labels). registerProduct still requires a
// `headerCentreComponent`, so this component exists but renders nothing.
import type React from 'react';

export const InsightsHeader: React.FC = () => null;
