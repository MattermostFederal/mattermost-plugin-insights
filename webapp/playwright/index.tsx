// Playwright component-test setup. Wraps every mounted component in
// `IntlProvider` (so `<FormattedMessage>` / `useIntl()` work without a
// host) and a Redux `Provider` (so `useDispatch` / `useSelector` don't
// throw). Translations come from the same en.json the plugin ships.
//
// Tests can pass a custom Redux state via `mount(<C/>, {hooksConfig:
// {state: {...}}})` to populate the shape `useSelector` reads — useful
// for license / theme / current-team / current-user fixtures.

import {beforeMount} from '@playwright/experimental-ct-react/hooks';
import {IntlProvider} from 'react-intl';
import {Provider} from 'react-redux';
import {legacy_createStore as createStore, applyMiddleware} from 'redux';
import {thunk} from 'redux-thunk';

import en from '../src/i18n/en.json';

export interface CTHooksConfig {
    state?: Record<string, unknown>;
}

const defaultStore = createStore(() => ({}), applyMiddleware(thunk));

beforeMount<CTHooksConfig>(async ({App, hooksConfig}) => {
    const store = hooksConfig?.state ?
        createStore(() => hooksConfig.state ?? {}, applyMiddleware(thunk)) :
        defaultStore;
    return (
        <Provider store={store}>
            <IntlProvider
                locale='en'
                messages={en}
            >
                <App/>
            </IntlProvider>
        </Provider>
    );
});
