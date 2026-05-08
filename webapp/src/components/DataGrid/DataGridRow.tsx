// Ported from mattermost/mattermost
// webapp/channels/src/components/admin_console/data_grid/data_grid_row.tsx
// (commit 26617fcbdc).

import React from 'react';
import type {CSSProperties} from 'react';

import type {Column, Row} from './DataGrid';

interface Props {
    columns: Column[];
    row: Row;
}

export const DataGridRow: React.FC<Props> = ({row, columns}) => {
    const cells = columns.map((column) => {
        const style: CSSProperties = {};
        if (column.width) {
            style.flexGrow = column.width;
        }
        if (column.textAlign) {
            style.textAlign = column.textAlign;
        }
        if (column.overflow) {
            style.overflow = column.overflow as CSSProperties['overflow'];
        }
        return (
            <div
                key={column.field}
                className={`DataGrid_cell${column.className ? ` ${column.className}` : ''}`}
                style={style}
            >
                {row.cells[column.field]}
            </div>
        );
    });
    return (
        <div
            className='DataGrid_row'
            onClick={row.onClick}
            role={row.onClick ? 'button' : undefined}
            tabIndex={row.onClick ? 0 : undefined}
        >
            {cells}
        </div>
    );
};
