import manifest from 'manifest';

import type {PluginRegistry} from 'types/mattermost-webapp';

import {installClient4Shim} from './client/Client4Shim';
import {InsightsHeader} from './components/InsightsHeader';
import {InsightsProduct} from './components/InsightsProduct';
import {getTranslationsForLocale} from './i18n';
import reducer from './redux/reducer';

import './styles/insights.scss';

export default class Plugin {
    public async initialize(registry: PluginRegistry) {
        registry.registerReducer(reducer);
        registry.registerTranslations(getTranslationsForLocale);
        installClient4Shim();
        registry.registerProduct(
            '/insights',
            'chart-line',
            'Insights',
            '/insights',
            InsightsProduct,
            InsightsHeader,
            () => null,
            true,
        );
    }
}

declare global {
    interface Window {
        registerPlugin(pluginId: string, plugin: Plugin): void;
    }
}

window.registerPlugin(manifest.id, new Plugin());
