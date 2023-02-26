<template>
  <div ref="main" class="page-profiler">
    <main class="page-profiler__main">
      <section ref="calls" class="page-profiler__callstack">
        <PerfectScrollbar :style="{ height: '100vh' }">
          <div class="page-profiler__callstack-items">
            <div class="page-profiler__callstack-items-top">
              <div>CPU / Memory</div>
              <div>Calls</div>
            </div>

            <div
              v-for="(edge, key) in sortedEdges"
              :key="key"
              class="page-profiler__callstack-item"
              @mouseover="setActiveEdge($event, edge)"
              @mouseout="setActiveEdge(null)"
            >
              <div class="page-profiler__callstack-item-usage">
                <div :style="calcStyles(edge.cost.p_cpu)" />
                <div :style="calcStyles(edge.cost.p_mu)" />
                <div>{{ normalizeCostValue(edge) }}</div>
              </div>

              <div class="page-profiler__callstack-item-calls">
                {{ edge.cost.ct }}
              </div>
            </div>
          </div>
        </PerfectScrollbar>
      </section>

      <div ref="info" class="page-profiler__stat">
        <section class="page-profiler__stat-board">
          <stat-board :cost="event.payload.peaks" />
        </section>

        <section class="page-profiler__stat-tabs">
          <!--        <Tabs class="flex-1">-->
          <!--          <Tab title="Call graph" class="flex-1">-->
          <!--            <Graph :event="event" @hover="showEdge" @hide="hideEdge"/>-->
          <!--          </Tab>-->
          <!--          <Tab title="Flamechart" class="flex-1">-->
          <!--            <FlameGraph :event="event" :width="width" @hover="showEdge" @hide="hideEdge"/>-->
          <!--          </Tab>-->
          <!--        </Tabs>-->
        </section>
      </div>

      <div
        v-if="activeEdge"
        class="page-profiler__edge"
        :style="activeEdgeStyle"
      >
        <h4 class="page-profiler__edge-title">
          {{ activeEdge.callee }}
        </h4>

        <stat-board :cost="activeEdge.cost" />
      </div>
    </main>
  </div>
</template>

<script lang="ts">
import { defineComponent, PropType } from "vue";
import { NormalizedEvent } from "~/config/types";
import StatBoard from "~/components/StatBoard/StatBoard.vue";
import { PerfectScrollbar } from "vue3-perfect-scrollbar";
import type { Profiler, ProfilerEdge } from "~/config/types";

export default defineComponent({
  components: {
    StatBoard,
    PerfectScrollbar,
  },
  props: {
    event: {
      type: Object as PropType<NormalizedEvent>,
      required: true,
    },
  },
  data() {
    return {
      callStackHeight: "0",
      activeEdge: null,
      activeEdgePosition: {
        x: 0,
        y: 0,
      },
    };
  },
  computed: {
    sortedEdges() {
      return Object.fromEntries(
        Object.entries((this.event.payload as Profiler).edges).sort(
          ([, a], [, b]) => b.cost.p_cpu - a.cost.p_cpu
        )
      );
    },
    activeEdgeStyle() {
      const width = 750;
      const height = 150;

      let top = this.activeEdgePosition.y;
      let left = this.activeEdgePosition.x;

      if (width + this.activeEdgePosition.x > window.innerWidth - 80) {
        const deltaX =
          width + this.activeEdgePosition.x - window.innerWidth + 100;
        left -= deltaX;
      }

      if (height + this.activeEdgePosition.y > window.innerHeight) {
        top = this.activeEdgePosition.y - height;
      }

      return {
        top: `${top + 10}px`,
        left: `${left}px`,
        width: `${width}px`,
      };
    },
  },
  methods: {
    calcStyles(percentages: number) {
      return {
        width: percentages
          ? `${this.normalizePercentage(percentages)}%`
          : `0px`,
      };
    },
    setActiveEdge(event: MouseEvent, edge: ProfilerEdge) {
      this.activeEdge = edge;
      this.activeEdgePosition = {
        x: event?.pageX || 0,
        y: event?.pageY || 0,
      };
    },
    normalizePercentage(n: number) {
      return Math.min(100, n);
    },
    normalizeCostValue(edge: ProfilerEdge) {
      return `${edge.cost.p_cpu}% / ${edge.cost.p_mu}%`;
    },
  },
});
</script>

<style lang="scss" scoped>
.page-profiler {
  @apply relative;
}

.page-profiler__main {
  @apply flex flex-col md:flex-row;
}
.page-profiler__callstack {
  @apply w-full md:w-1/6 border-r border-gray-300 dark:border-gray-500;
}

.page-profiler__stat {
  @apply w-full h-full flex flex-col md:w-5/6 divide-y divide-gray-300 dark:divide-gray-500;
}

.page-profiler__stat-board {
  @apply p-5 bg-gray-200 bg-gray-800;
}

.page-profiler__stat-tabs {
  @apply p-5 bg-gray-200 flex-1 flex flex-col bg-gray-800;
}

.page-profiler__callstack-items {
}

.page-profiler__callstack-items-top {
  @apply flex items-stretch justify-between bg-gray-600 border-b border-gray-600 text-xs text-center py-2 text-gray-300;

  *:first-child {
    @apply flex-1 border-r border-gray-600;
  }

  *:last-child {
    @apply w-12;
  }
}

.page-profiler__callstack-item {
  @apply flex items-stretch border-b border-gray-600 hover:bg-gray-900 dark:hover:bg-white cursor-pointer;
}

.page-profiler__callstack-item-usage {
  @apply flex-1 text-center text-xs relative border-r border-gray-600;

  *:first-child {
    @apply h-full bg-red-800 text-sm opacity-70;
  }

  *:nth-child(2) {
    @apply h-full bg-purple-800 text-sm opacity-40 -mt-6;
  }

  *:nth-child(3) {
    @apply absolute inset-0 py-1;
  }
}

.page-profiler__callstack-item-calls {
  @apply w-12 text-center text-xs py-1;

  .page-profiler__callstack-item:hover & {
    @apply text-gray-300;
  }
}

.page-profiler__edge {
  @apply bg-gray-800 absolute border border-gray-300 dark:border-gray-600;

  z-index: 9999;
}

.page-profiler__edge-title {
  @apply px-4 py-2 font-bold truncate text-gray-300;
}
</style>
