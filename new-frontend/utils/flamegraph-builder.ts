import { ProfilerEdges } from "~/config/types";
import {FlameChartNode} from "flame-chart-js/dist/types";

export default class {
  private walked: string[] = []

  private readonly edges: ProfilerEdges

  constructor(edges: ProfilerEdges) {
    this.edges = edges
  }

  build(field = 'cpu'): FlameChartNode {
    this.walked = []

    const datum: Record<string, FlameChartNode> = {}

    Object.values(this.edges).forEach((edge) => {
      const parent = edge.caller
      const target = edge.callee

      const duration = (edge.cost[field] || 0) > 0 ? edge.cost[field] / 1_000 : 0
      const start = 0

      if (target && !datum[target]) {
        datum[target] = {
          name: target,
          start,
          duration,
          // cost: edge.cost,
          children: []
        }
      }

      if (parent && !datum[parent]) {
        datum[parent] = {
          name: parent,
          start,
          duration,
          // cost: edge.cost,
          children: []
        }
      }

      // NOTE: walked skips several targettions (recursion detected), should be fixed
      if (!parent || this.walked.includes(target)) {
        // console.log(node, target)
        return
      }

      if (datum[parent] && datum[parent].children) {
        const parentChildren = datum[parent].children || []

        const lastChild = parentChildren ? parentChildren[parentChildren.length - 1]: null
        datum[target].start = lastChild ? lastChild.start + lastChild.duration : datum[target].start
      } else {
        datum[target].start += datum[parent].start
      }

      datum[parent].children?.push(datum[target])
      this.walked.push(target)
    })

    this.walked = []

    return datum['main()']
  }
}
