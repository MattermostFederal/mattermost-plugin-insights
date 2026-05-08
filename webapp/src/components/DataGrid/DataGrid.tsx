// Ported from mattermost/mattermost
// webapp/channels/src/components/admin_console/data_grid/data_grid.tsx
// (commit 26617fcbdc).
//
// Adaptations:
//   - The deprecated DataGrid included an optional search bar, filter
//     popover, and admin-console-shaped loading spinner. None of those
//     are used by the insights tables, so they are stripped here. Search
//     / filter can be reintroduced verbatim from the deprecated source if
//     a future insight needs them.
//   - LoadingSpinner / Next / Previous icons → host icon font.
//   - FormattedMessage → plain text (i18n keys land in section I).

import React, {useEffect, useRef, useState} from 'react';
import type {CSSProperties} from 'react';

import {DataGridHeader} from './DataGridHeader';
import {DataGridRow} from './DataGridRow';

export interface Column {
    name: string | React.ReactNode;
    field: string;
    fixed?: boolean;
    className?: string;
    width?: number;
    textAlign?: CSSProperties['textAlign'];
    overflow?: string;
}

export interface Row {
    cells: Record<string, React.ReactNode>;
    onClick?: () => void;
}

interface Props {
    rows: Row[];
    columns: Column[];
    placeholderEmpty?: React.ReactNode;
    loadingIndicator?: React.ReactNode;

    rowsContainerStyles?: CSSProperties;
    minimumColumnWidth?: number;

    startCount: number;
    endCount: number;
    total?: number;
    loading: boolean;

    nextPage: () => void;
    previousPage: () => void;

    className?: string;
}

const MINIMUM_COLUMN_WIDTH = 100;

export const DataGrid: React.FC<Props> = ({
    rows,
    columns,
    placeholderEmpty,
    loadingIndicator,
    rowsContainerStyles,
    minimumColumnWidth = MINIMUM_COLUMN_WIDTH,
    startCount,
    endCount,
    total,
    loading,
    nextPage,
    previousPage,
    className,
}) => {
    const ref = useRef<HTMLDivElement>(null);
    const [visibleColumns, setVisibleColumns] = useState<Column[]>(columns);

    useEffect(() => {
        const handleResize = () => {
            if (!ref.current) {
                return;
            }
            const fixedColumns = columns.filter((c) => c.fixed);
            const fixedColumnWidth = fixedColumns.length * minimumColumnWidth;
            let availableWidth = ref.current.clientWidth - fixedColumnWidth - 50;
            const next = columns.filter((column) => {
                if (availableWidth > minimumColumnWidth) {
                    availableWidth -= minimumColumnWidth;
                    return true;
                }
                return Boolean(column.fixed);
            });
            setVisibleColumns(next);
        };
        handleResize();
        window.addEventListener('resize', handleResize);
        return () => window.removeEventListener('resize', handleResize);
    }, [columns, minimumColumnWidth]);

    let body: React.ReactNode;
    if (loading) {
        body = (
            <div className='DataGrid_loading'>
                {loadingIndicator ?? 'Loading…'}
            </div>
        );
    } else if (rows.length === 0) {
        body = (
            <div className='DataGrid_empty'>
                {placeholderEmpty ?? 'No items found'}
            </div>
        );
    } else {
        body = rows.map((row, index) => (
            <DataGridRow
                key={index}
                row={row}
                columns={visibleColumns}
            />
        ));
    }

    const renderFooter = () => {
        if (!total) {
            return null;
        }
        const firstPage = startCount <= 1;
        const lastPage = endCount >= total;
        return (
            <div className='DataGrid_footer'>
                <div className='DataGrid_cell'>
                    <span>{`${startCount} - ${endCount} of ${total}`}</span>
                    <button
                        type='button'
                        aria-label='Previous page'
                        className={'btn btn-quaternary btn-icon btn-sm ml-2 prev ' + (firstPage ? 'disabled' : '')}
                        onClick={loading || firstPage ? undefined : previousPage}
                        disabled={firstPage}
                    >
                        <i className='icon icon-chevron-left'/>
                    </button>
                    <button
                        type='button'
                        aria-label='Next page'
                        className={'btn btn-quaternary btn-icon btn-sm next ' + (lastPage ? 'disabled' : '')}
                        onClick={loading || lastPage ? undefined : nextPage}
                        disabled={lastPage}
                    >
                        <i className='icon icon-chevron-right'/>
                    </button>
                </div>
            </div>
        );
    };

    return (
        <div
            className={`DataGrid${className ? ` ${className}` : ''}`}
            ref={ref}
        >
            <DataGridHeader columns={visibleColumns}/>
            <div
                className='DataGrid_rows'
                style={rowsContainerStyles}
            >
                {body}
            </div>
            {renderFooter()}
        </div>
    );
};
