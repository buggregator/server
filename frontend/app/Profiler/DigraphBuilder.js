import {humaCpuUsage, humanFileSize, formatDuration} from "@/Utils/converters"

const labelsStrigifier = function (labels) {
  return Object.entries(labels).map(([label, value]) => {
    return `${label}="${value}"`
  }).join(' ')
}

const metricName = function (metric) {
  switch (metric) {
    case 'cpu':
    case 'p_cpu':
      return 'CPU'
    case 'wt':
      return 'Wall time'
    case 'mu':
    case 'p_mu':
      return 'Memory'
    case 'pmu':
      return 'Memory leak'
  }
}

const formatValue = function (value, metric) {
  switch (metric) {
    case 'p_mu':
    case 'p_pmu':
    case 'p_cpu':
      return `${value}%`
    case 'mu':
    case 'pmu':
      return humanFileSize(value)
    case 'cpu':
    case 'wt':
      return formatDuration(value)
  }

  return value
}

const generateNode = function (edge, metric) {
  let parent = edge.caller || ''
  let func = edge.callee || ''

  let labels = {
    label: ` ${metricName(metric)} - ${formatValue(edge.cost[metric], metric)} [${edge.cost.ct}x]`,
    fontsize: '10px'
  }

  return `    "${parent}" -> "${func}" [ ${labelsStrigifier(labels)} ]\n`
}

export default class {
  constructor(edges) {
    this.edges = edges
  }

  build(metric = "p_cpu", threshold = 1) {
    let digram = `
digraph xhprof {
    splines=true;
    overlap=false;
    node [ shape="box" style="filled" fontname="Arial" margin=0.2 ]
    edge [ fontname="Arial" ]
`

    let types = {
      pmu: {
        node: {
          color: '#891d1d',
          fontcolor: '#FFFFFF'
        },
        edge: {
          fontcolor: '#891d1d',
          color: '#ED96AC',
        },
        nodes: []
      },

      default: {
        node: {
          color: '#EEEEEE',
          fontcolor: '#666666'
        },
        edge: {
          color: '#999999',
        },
        nodes: []
      },
    }

    const edges = Object.entries(this.edges)

    for (const [key, edge] of edges) {
      if (edge.cost.pmu > 0) {
        types.pmu.nodes.push([edge, 'pmu'])
      } else if (edge.cost[metric] >= threshold) {
        types.default.nodes.push([edge, metric])
      }
    }

    const nodes = Object.entries(types)

    for (const [key, config] of nodes) {
      digram += `    node [ ${labelsStrigifier(config.node)} ]\n`
      digram += `    edge [ ${labelsStrigifier(config.edge)} ]\n`

      for (let [node, metric] of config.nodes) {
        digram += generateNode(node, metric)
      }
    }

    return `${digram} }`
  }
}
