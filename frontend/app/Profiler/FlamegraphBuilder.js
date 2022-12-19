export default class {
  walked = []

  constructor(edges) {
    this.edges = edges
  }

  build() {
    this.walked = []

    let datum = {
      0: {name: '_', value: 0, children: []}
    }

    const edges = Object.entries(this.edges)

    for (const [key, edge] of edges) {
      let parent = edge.caller || null
      let func = edge.callee || null

      let value = edge.cost.cpu || 0

      if (!datum.hasOwnProperty(func)) {
        datum[func] = {
          name: func,
          value: value,
          cost: edge.cost,
          children: []
        }
      }

      if (parent && !datum.hasOwnProperty(parent)) {
        datum[parent] = {
          name: parent,
          value: value,
          cost: edge.cost,
          children: []
        }
      }

      let node = parent || null

      // TODO walked skips several functions (recursion detected), should be fixed
      if (!node || this.walked.includes(func)) {
        // console.log(node, func)
        continue
      }

      datum[node].children.push(datum[func])
      this.walked.push(func)
    }

    this.walked = []

    return datum['main()']
  }
}
