// Ported from mattermost/mattermost
// webapp/channels/src/components/activity_and_insights/insights/time_frame_dropdown/time_frame_dropdown.tsx
// (commit 26617fcbdc).
//
// Adaptations:
//   - The deprecated source uses ChevronDownIcon from `@mattermost/compass-icons`.
//     We reuse the host's bundled icon font instead (`<i className='icon icon-chevron-down'/>`)
//     so we don't depend on a typed-React-18 mismatch in compass-icons.
//   - react-select 5.x's `StateManagedSelect` type is rejected by React 18's
//     JSX element type, so we cast the default export through `unknown`.

import React, {memo} from 'react';
import ReactSelectImport, {components as selectComponents} from 'react-select';

import type {TimeRange} from '../../types';

interface SelectOption {
    value: TimeRange;
    label: string;
}

interface Props {
    value: TimeRange;
    onChange: (range: TimeRange) => void;
}

// 'Today' was replaced by 'Yesterday': a closed, complete-day window. The
// label says Yesterday rather than "last 24 hours" because that is literally
// what it covers — today's activity is not in the snapshot at all.
// See webapp/src/types.ts.
const options: SelectOption[] = [
    {value: '1_day', label: 'Yesterday'},
    {value: '7_day', label: 'Last 7 days'},
    {value: '28_day', label: 'Last 28 days'},
];

const reactStyles = {
    control: (provided: React.CSSProperties) => ({
        ...provided,
        width: '140px',
        cursor: 'pointer',
        fontSize: '12px',
        lineHeight: '16px',
    }),
    indicatorSeparator: (provided: React.CSSProperties) => ({
        ...provided,
        display: 'none',
    }),
    option: (provided: React.CSSProperties) => ({
        ...provided,
        cursor: 'pointer',
    }),
    menuPortal: (provided: React.CSSProperties) => ({
        ...provided,
        zIndex: 1100,
    }),
};

const DropdownIndicatorBase = selectComponents.DropdownIndicator as unknown as React.ComponentType<Record<string, unknown>>;
const DropdownIndicator: React.FC<Record<string, unknown>> = (props) => (
    <DropdownIndicatorBase {...props}>
        <span className='icon'>
            <i className='icon icon-chevron-down'/>
        </span>
    </DropdownIndicatorBase>
);

const ReactSelect = ReactSelectImport as unknown as React.ComponentType<Record<string, unknown>>;

const TimeRangeSelectComponent: React.FC<Props> = ({value, onChange}) => {
    const current = options.find((o) => o.value === value) ?? options[0];

    const handleChange = (selected: SelectOption | null) => {
        if (selected) {
            onChange(selected.value);
        }
    };

    return (
        <ReactSelect
            className='insights-time-range'
            classNamePrefix='insights-time-range'
            inputId='insightsTemporal'
            menuPortalTarget={typeof document === 'undefined' ? null : document.body}
            styles={reactStyles}
            options={options}
            isClearable={false}
            onChange={handleChange}
            value={current}
            aria-labelledby='changeInsightsTemporal'
            components={{DropdownIndicator}}
            isSearchable={false}
        />
    );
};

export const TimeRangeSelect = memo(TimeRangeSelectComponent);
