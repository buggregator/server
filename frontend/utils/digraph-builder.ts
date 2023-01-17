import {humanFileSize, formatDuration} from "~/utils/formats";
import {ProfilerEdge, ProfilerEdges} from "~/config/types";

const labelsStrigifier = (labels: object): string => Object.entries(labels)
    .map(([label, value]) => `${label}="${value}"`)
    .join(' ')

const formatValue = (value: number, metric: string): string | number => {
    switch (metric) {
        case 'p_mu':
        case 'p_pmu':
        case 'p_cpu':
        case 'p_wt':
            return `${value}%`
        case 'mu':
        case 'd_mu':
        case 'pmu':
        case 'd_pmu':
            return humanFileSize(value)
        case 'cpu':
        case 'd_cpu':
        case 'wt':
        case 'd_wt':
            return formatDuration(value)
        default:
            return value
    }
}

export const addSlashes = (str: string): string => str.replace(/\\/g, '\\\\');

const generateNode = (edge: ProfilerEdge, metric: { field: string, label: string }): string => {
    const parent = addSlashes(edge.caller || '')
    const func = addSlashes(edge.callee || '')

    let label = formatValue(edge.cost[metric.field], metric.label)
    if (edge.cost.ct > 1) {
        label += ` - ${edge.cost.ct}x`
    }

    const labels = {
        label,
    }

    return `    "${parent}" -> "${func}" [ ${labelsStrigifier(labels)} ]\n`
}

export class DigraphBuilder {
    private readonly edges: ProfilerEdges;

    constructor(edges: ProfilerEdges) {
        this.edges = edges
    }

    build(metric = "cpu", threshold = 1) {
        let digram = `
digraph xhprof {
    splines=true;
    overlap=false;
    node [ shape="box" style="rounded" fontname="Arial" margin=0.3 ]
    edge [ fontname="Arial" ]
`

        let metricProps = {field: 'p_cpu', label: 'p_cpu'}
        switch (metric) {
            case 'pmu':
                metricProps = {field: 'p_pmu', label: 'p_pmu'}
                break;
            case 'mu':
                metricProps = {field: 'p_mu', label: 'p_mu'}
                break;
            default:
                break;
        }

        const types = {
            pmu: {
                node: {
                    class: 'pmu',
                },
                edge: {
                    // fontcolor: '#891d1d',
                    color: '#ED96AC',
                },
                nodes: []
            },

            default: {
                node: {
                    class: 'default',
                },
                edge: {
                    color: '#999999',
                },
                nodes: []
            },
        }

        const edges = Object.entries(this.edges)

        // eslint-disable-next-line no-restricted-syntax
        for (const [, edge] of edges) {
            if (!edge.caller || edge.caller.length === 0) {
                // eslint-disable-next-line no-continue
                continue;
            }

            if (edge.cost.p_pmu > 10) {
                types.pmu.nodes.push([edge, metricProps])
            } else if (edge.cost[metricProps.field] >= threshold) {
                types.default.nodes.push([edge, metricProps])
            }
        }

        const nodes = Object.entries(types)

        // eslint-disable-next-line no-restricted-syntax
        for (const [, config] of nodes) {
            digram += `    node [ ${labelsStrigifier(config.node)} ]\n`
            digram += `    edge [ ${labelsStrigifier(config.edge)} ]\n`

            // eslint-disable-next-line no-restricted-syntax
            for (const [n, m] of config.nodes) {
                digram += generateNode(n, m)
            }
        }

        return `${digram} }`
    }
}
