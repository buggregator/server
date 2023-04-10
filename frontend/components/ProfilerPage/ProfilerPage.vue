<template>
  <div class="profiler-page">
    <div class="profiler-page__head"></div>
    <main class="profiler-page__main">
      <section ref="calls" class="profiler-page__callstack">
        <PerfectScrollbar :style="{ height: '100vh' }">
          <ProfilerPageCallStack
            :event="event.payload"
            @hover="setActiveEdge"
            @hide="setActiveEdge"
          />
        </PerfectScrollbar>
      </section>

      <div ref="info" class="profiler-page__stat">
        <section class="profiler-page__stat-board">
          <StatBoard :cost="event.payload.peaks" />
        </section>

        <section class="profiler-page__stat-tabs">
          <Tabs :options="{ useUrlFragment: false }" @changed="tabChange">
            <Tab name="Call graph">
              <ProfilerPageCallGraph
                :event="event.payload"
                @hover="setActiveEdge"
                @hide="setActiveEdge"
              />
            </Tab>
            <Tab name="Flamechart">
              <ProfilePageFlamegraph
                :key="activeTab"
                :data-key="activeTab"
                :edges="event.payload.edges"
                @hover="setActiveEdge"
                @hide="setActiveEdge"
              />
            </Tab>
          </Tabs>
        </section>
      </div>

      <div
        v-if="activeEdge"
        class="profiler-page__edge"
        :style="activeEdgeStyle"
      >
        <h4 class="profiler-page__edge-title">
          {{ activeEdge.callee }}
        </h4>

        <StatBoard :cost="activeEdge.cost" />
      </div>
    </main>
  </div>
</template>

<script lang="ts">
import { defineComponent, PropType } from "vue";
import { NormalizedEvent } from "~/config/types";
import StatBoard from "~/components/StatBoard/StatBoard.vue";
import ProfilerPageCallStack from "~/components/ProfilerPageCallStack/ProfilerPageCallStack.vue";
import ProfilerPageCallGraph from "~/components/ProfilerPageCallGraph/ProfilerPageCallGraph.vue";
import ProfilePageFlamegraph from "~/components/ProfilePageFlamegraph/ProfilePageFlamegraph.vue";
import { PerfectScrollbar } from "vue3-perfect-scrollbar";
import type { Profiler, ProfilerEdge } from "~/config/types";
import { Tabs, Tab } from "vue3-tabs-component";

export default defineComponent({
  components: {
    StatBoard,
    ProfilerPageCallStack,
    ProfilerPageCallGraph,
    ProfilePageFlamegraph,
    PerfectScrollbar,
    Tabs,
    Tab,
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
      activeTab: "",
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
    calcStyles(percentages: number): { width: string } {
      return {
        width: percentages
          ? `${this.normalizePercentage(percentages)}%`
          : `0px`,
      };
    },
    setActiveEdge(edge): void {
      this.activeEdge = edge;
      this.activeEdgePosition = edge?.position;
    },
    normalizePercentage(n: number): number {
      return Math.min(100, n);
    },
    normalizeCostValue(edge: ProfilerEdge): string {
      return `${edge.cost.p_cpu}% / ${edge.cost.p_mu}%`;
    },
    tabChange(selectedTab: { tab: { name: string } }) {
      this.activeTab = selectedTab.tab.name;
    },
  },
});
</script>

<style lang="scss" scoped>
.profiler-page {
  @apply relative;
}

.profiler-page__main {
  @apply flex flex-col md:flex-row;
}

.profiler-page__callstack {
  @apply w-full md:w-1/5 border-r border-gray-300 dark:border-gray-500;
}

.profiler-page__stat {
  @apply w-full flex flex-col md:w-4/5 divide-y divide-gray-300 dark:divide-gray-500;
}

.profiler-page__stat-board {
  @apply bg-gray-200 dark:bg-gray-800;
}

.profiler-page__stat-tabs {
  @apply p-5 bg-gray-200 flex-1 flex flex-col dark:bg-gray-800 dark:text-gray-300;
}

.profiler-page__stat-tabs .tabs-component-panel {
  @apply h-full;
}

.profiler-page__callstack-items {
  @apply flex-row;
}

.profiler-page__callstack-items-top {
  @apply flex items-stretch justify-between bg-gray-600 border-b border-gray-600 text-xs text-center py-2 text-gray-300;

  *:first-child {
    @apply flex-1 border-r border-gray-600;
  }

  *:last-child {
    @apply w-12;
  }
}

.profiler-page__callstack-item {
  @apply flex items-stretch border-b border-gray-600 hover:bg-gray-900 dark:hover:bg-white cursor-pointer;
}

.profiler-page__callstack-item-usage {
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

.profiler-page__callstack-item-calls {
  @apply w-12 text-center text-xs py-1;

  .profiler-page__callstack-item:hover & {
    @apply text-gray-300;
  }
}

.profiler-page__edge {
  @apply bg-gray-800 absolute border border-gray-300 dark:border-gray-600;

  z-index: 9999;
}

.profiler-page__edge-title {
  @apply px-4 py-2 font-bold truncate text-gray-300;
}
</style>
