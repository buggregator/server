<template>
  <div class="flex justify-items-stretch items-stretch bg-white" style="height: 500px">
  </div>
</template>

<script>
import {wasmFolder} from "@hpcc-js/wasm";
import {select} from "d3-selection"
import {graphviz} from "d3-graphviz"
import {humanFileSize, formatDuration} from "@/Utils/converters"

wasmFolder("https://cdn.jsdelivr.net/npm/@hpcc-js/wasm/dist");

const labelsStrigifier = function (labels) {
  return Object.entries(labels).map(([label, value]) => {
    return `${label}="${value}"`
  }).join(' ')
}

export default {
  props: {
    event: Object
  },
  methods: {
    buildDigram() {
      let digram = `
digraph xhprof {
    rankdir="TB";
    splines=true;
    overlap=false;
    nodesep="0.2";
    ranksep="0.4";
    labelloc="t";
    node [ shape="box" style="filled" fontname="Arial" margin=0.2 ]
    edge [ fontname="Arial" ]
`
      const edges = Object.entries(this.event.edges)
      const peak = this.event.peaks

      for (const [key, edge] of edges) {
        let parent = edge.caller || ''
        let func = edge.callee || ''

        let labels = {label: ` CPU: ${edge.cost.p_cpu}% [${edge.cost.ct}x]`}
        if (edge.cost.pmu > 0) {
          labels['label'] = labels['label'] + ' PMU: ' + edge.cost.pmu + '%'
        }

        if (edge.cost.pmu > 0 || edge.cost.p_cpu >= 1) {
          digram += `    "${parent}" -> "${func}" [ ${labelsStrigifier(labels)} ]\n`
        }
      }

      return `${digram} }`
    }
  },
  mounted() {
    select(this.$el)
      .graphviz()
      .width('100%')
      .height('100%')
      .fit(true)
      .renderDot(this.buildDigram());
  }
}
</script>
