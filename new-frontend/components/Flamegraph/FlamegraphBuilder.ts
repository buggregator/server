import {ProfilerEdge} from "~/config/types";
import {FlameChartNode} from "flame-chart-js/dist/types";

export default class {
  private walked = []

  private readonly edges: ProfilerEdge[]

  constructor(edges: ProfilerEdge[]) {
    this.edges = edges
  }

  build(field = 'cpu'): Array<FlameChartNode> {
    this.walked = []

    const datum = {
      0: {name: '_', value: 0, children: []}
    }

    const edges = Object.entries(this.edges)

    for (const [key, edge] of edges) {
      const parent = edge.caller || ''
      const func = edge.callee || ''

      const duration = (edge.cost[field] || 0) > 0 ? edge.cost[field] / 1_000 : 0
      const start = 0

      if (!datum.hasOwnProperty(func)) {
        datum[func] = {
          name: func,
          start,
          duration,
          cost: edge.cost,
          children: []
        }
      }

      if (parent && !datum.hasOwnProperty(parent)) {
        datum[parent] = {
          name: parent,
          start,
          duration,
          cost: edge.cost,
          children: []
        }
      }

      const node = parent || null

      // TODO walked skips several functions (recursion detected), should be fixed
      if (!node || this.walked.includes(func)) {
        // console.log(node, func)
        continue
      }

      if (datum[node].children.length > 0) {
        const lastChild = datum[node].children[datum[node].children.length - 1]
        datum[func].start = lastChild.start + lastChild.duration
      } else {
        datum[func].start += datum[node].start
      }

      datum[node].children.push(datum[func])
      this.walked.push(func)
    }

    this.walked = []

    return datum['main()']
  }
}
