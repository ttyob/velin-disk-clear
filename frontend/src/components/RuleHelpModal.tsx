import {AlertTriangle, CircleHelp, ExternalLink, ShieldAlert, X} from 'lucide-react';
import type {rules, scanner} from '../../wailsjs/go/models';
import {api} from '../lib/api';
import {formatBytes, recommendationLabels, riskLabels} from '../lib/format';

type HelpSource = rules.Rule | scanner.Item;

interface Props {
    source: HelpSource | null;
    onClose: () => void;
}

function isItem(source: HelpSource): source is scanner.Item {
    return 'rule_id' in source;
}

export function RuleHelpModal({source, onClose}: Props) {
    if (!source) return null;
    const item = isItem(source);
    const name = item ? source.name : source.name;
    const ruleID = item ? source.rule_id : source.id;
    const help = item ? {
        summary: source.help_summary,
        details: source.help_details,
        warning: source.special_warning,
        steps: source.manual_steps,
    } : {
        summary: source.help.summary,
        details: source.help.details,
        warning: source.help.special_warning,
        steps: source.help.manual_steps,
    };
    const risk = source.risk;
    const recommendation = source.recommendation;
    const defaultSelected = source.default_selected;
    const purpose = source.purpose;
    const effect = source.clean_effect;
    const isPagefile = ruleID === 'windows.pagefile_analysis';

    return (
        <div className="modal-backdrop" role="presentation" onMouseDown={onClose}>
            <section className="modal help-modal" role="dialog" aria-modal="true" aria-labelledby="help-title" onMouseDown={event => event.stopPropagation()}>
                <header className="modal-header">
                    <div className="modal-title">
                        <span className="icon-box"><CircleHelp size={18}/></span>
                        <div>
                            <h2 id="help-title">{name}</h2>
                            <p>{ruleID}</p>
                        </div>
                    </div>
                    <button className="icon-button" aria-label="关闭" title="关闭" onClick={onClose}><X size={18}/></button>
                </header>

                <div className="help-summary">{help.summary}</div>
                {help.warning && (
                    <div className="warning-callout">
                        <ShieldAlert size={18}/>
                        <div><strong>特别提醒</strong><p>{help.warning}</p></div>
                    </div>
                )}

                <dl className="detail-list">
                    <div><dt>当前项用途</dt><dd>{purpose}</dd></div>
                    <div><dt>清理后影响</dt><dd>{effect}</dd></div>
                    <div><dt>处理建议</dt><dd>{recommendationLabels[recommendation] ?? recommendation}</dd></div>
                    <div><dt>风险等级</dt><dd><span className={`risk-badge risk-${risk}`}>{riskLabels[risk] ?? risk}</span></dd></div>
                    <div><dt>默认选中</dt><dd>{defaultSelected ? '是' : '否'}</dd></div>
                    {item && <div><dt>磁盘占用</dt><dd>{formatBytes(source.allocated_size)}</dd></div>}
                    {item && <div><dt>文件位置</dt><dd className="path-value">{source.path}</dd></div>}
                </dl>

                {help.details && <div className="help-details"><h3>详细说明</h3><p>{help.details}</p></div>}
                {help.steps && help.steps.length > 0 && (
                    <div className="manual-steps">
                        <h3>建议处理方式</h3>
                        <ol>{help.steps.map(step => <li key={step}>{step}</li>)}</ol>
                    </div>
                )}

                <footer className="modal-footer">
                    {isPagefile && (
                        <button className="button secondary" onClick={() => api.openSystemSettings('virtual_memory')}>
                            <ExternalLink size={16}/>打开虚拟内存设置
                        </button>
                    )}
                    {risk === 'high' && !isPagefile && <span className="footer-note"><AlertTriangle size={15}/>需要手动确认</span>}
                    <button className="button primary" onClick={onClose}>知道了</button>
                </footer>
            </section>
        </div>
    );
}
