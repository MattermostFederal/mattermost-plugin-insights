// Ported from mattermost/mattermost
// webapp/channels/src/components/admin_console/data_grid/data_grid_header.tsx
// (commit 26617fcbdc).

import React from 'react';
import type {CSSProperties} from 'react';

import type {Column} from './DataGrid';

interface Props {
    columns: Column[];
}

export const DataGridHeader: React.FC<Props> = ({columns}) => (
    <div className='DataGrid_header'>
        {columns.map((col) => {
            const style: CSSProperties = {};
            if (col.width) {
                style.flexGrow = col.width;
            }
            return (
                <div
                    key={col.field}
                    className='DataGrid_cell'
                    style={style}
                >
                    {col.name}
                </div>
            );
        })}
    </div>
);
