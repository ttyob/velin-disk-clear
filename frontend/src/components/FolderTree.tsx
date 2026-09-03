import {ChevronDown, ChevronRight, File, Folder as FolderIcon, HelpCircle} from 'lucide-react';
import {useEffect, useMemo, useRef, useState} from 'react';
import type {scanner} from '../../wailsjs/go/models';
import {formatBytes, riskLabels} from '../lib/format';

interface Props {
    folders: scanner.Folder[];
    items: scanner.Item[];
    selected: Set<string>;
    onToggleItem: (item: scanner.Item) => void;
    onToggleFolder: (items: scanner.Item[], selected: boolean) => void;
    onHelp: (item: scanner.Item) => void;
}

function Checkbox({checked, indeterminate, disabled, onChange, label}: {
    checked: boolean;
    indeterminate?: boolean;
    disabled?: boolean;
    onChange: (checked: boolean) => void;
    label: string;
}) {
    const ref = useRef<HTMLInputElement>(null);
    useEffect(() => {
        if (ref.current) ref.current.indeterminate = Boolean(indeterminate);
    }, [indeterminate]);
    return <input ref={ref} type="checkbox" checked={checked} disabled={disabled} aria-label={label} onChange={event => onChange(event.target.checked)}/>;
}

export function FolderTree(props: Props) {
    return <div className="folder-tree">{props.folders.map(folder => <FolderNode key={folder.id} folder={folder} depth={0} {...props}/>)}</div>;
}

function FolderNode({folder, depth, items, selected, onToggleItem, onToggleFolder, onHelp}: Props & {folder: scanner.Folder; depth: number}) {
    const [expanded, setExpanded] = useState(depth === 0);
    const matchingItems = useMemo(
        () => items.filter(item => folder.item_ids.includes(item.id)),
        [folder.item_ids, items],
    );
    const eligibleItems = matchingItems.filter(item => item.selectable && item.action !== 'analyze');
    const selectedEligible = eligibleItems.filter(item => selected.has(item.id)).length;
    const directItems = matchingItems.filter(item => normalise(item.directory) === normalise(folder.path));
    const checked = eligibleItems.length > 0 && selectedEligible === eligibleItems.length;
    const indeterminate = selectedEligible > 0 && !checked;
    const hasChildren = folder.children.length > 0 || directItems.length > 0;

    return (
        <div className="folder-node">
            <div className="folder-row" style={{paddingLeft: 12 + depth * 20}}>
                <button className="tree-toggle" disabled={!hasChildren} aria-label={expanded ? '收起文件夹' : '展开文件夹'} onClick={() => setExpanded(value => !value)}>
                    {hasChildren ? expanded ? <ChevronDown size={16}/> : <ChevronRight size={16}/> : <span/>}
                </button>
                <Checkbox
                    checked={checked}
                    indeterminate={indeterminate}
                    disabled={eligibleItems.length === 0}
                    label={`选择 ${folder.name} 中的可清理项`}
                    onChange={value => onToggleFolder(eligibleItems, value)}
                />
                <FolderIcon size={17} className="folder-icon"/>
                <span className="folder-name" title={folder.path}>{folder.name}</span>
                <span className="folder-count">{folder.file_count} 个文件</span>
                <span className={`risk-text risk-${folder.highest_risk}`}>{riskLabels[folder.highest_risk] ?? folder.highest_risk}</span>
                <strong>{formatBytes(folder.allocated_bytes)}</strong>
            </div>
            {expanded && (
                <>
                    {folder.children.map(child => (
                        <FolderNode
                            key={child.id}
                            folder={child}
                            depth={depth + 1}
                            items={items}
                            selected={selected}
                            onToggleItem={onToggleItem}
                            onToggleFolder={onToggleFolder}
                            onHelp={onHelp}
                            folders={[]}
                        />
                    ))}
                    {directItems.map(item => (
                        <div className="tree-file-row" style={{paddingLeft: 48 + depth * 20}} key={item.id}>
                            <input type="checkbox" checked={selected.has(item.id)} disabled={!item.selectable} aria-label={`选择 ${item.name}`} onChange={() => onToggleItem(item)}/>
                            <File size={15}/>
                            <span className="tree-file-name">{item.name}</span>
                            <span>{formatBytes(item.allocated_size)}</span>
                            <button className="icon-button subtle" title="查看清理建议" aria-label="查看清理建议" onClick={() => onHelp(item)}><HelpCircle size={15}/></button>
                        </div>
                    ))}
                </>
            )}
        </div>
    );
}

function normalise(value: string) {
    return value.replace(/[\\/]+$/, '').toLowerCase();
}
