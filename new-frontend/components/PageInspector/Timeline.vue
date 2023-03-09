<template>
  <section class="waterfall">
    <h3 class="waterfall-title">Timeline</h3>

    <div v-if="segmentTypes.length > 0" class="waterfall-segment-types">
      <div v-for="type in segmentTypes" class="waterfall-segment-type">
        <div :class="type.color" class="waterfall-segment-type__color-box"></div>
        <span class="waterfall-segment-type__label">{{ type.type }}</span>
      </div>
    </div>

    <div v-if="series.length > 0" class="waterfall-segments">
      <div class="waterfall-segments-cells">
        <div
            v-for="segment in grid.segments"
            :key="segment"
            class="waterfall-segments-cell">{{ segment }} ms
        </div>
      </div>
      <div class="waterfall-series" :style="{'background-size': `${grid.widthPercent}% 20%`}">
        <div v-for="row in series" class="waterfall-series__segment">
          <div
              class="waterfall-series__segment-label">
            {{ row.segment.label }} - {{ row.segment.duration }} ms
          </div>
          <div class="flex items-center w-full">
            <div :style="{'width': row.marginPercent + '%'}" class="waterfall-series__segment-start">
              <span class="waterfall-series__segment-start-label">{{ row.segment.start }} ms</span>
            </div>
            <div
                class="waterfall-series__segment-time" :class="[row.color]"
                :style="{width: row.widthPercent + '%'}"></div>
            <div class="waterfall-series__segment-end"></div>
          </div>
        </div>
      </div>
    </div>
    <div v-else class="flex w-full flex-col items-center mt-5">
      <div class="w-1/5">
        <HeartBeat class="text-blue-300"/>
      </div>
      <h3 class="text-lg md:text-xl lg:text-3xl mt-5 font-bold text-gray-300">No data</h3>
    </div>
  </section>
</template>

<script lang="ts">
import {defineComponent, PropType} from "vue";
import {InspectorSegment, InspectorTransaction, Inspector, NormalizedEvent} from "~/config/types";

const segmentColor = (color: string): string => {
  switch (color) {
    case 'sqlite':
      return 'orange';
    case 'view':
      return 'blue';
    case 'artisan':
      return 'purple';
    default:
      return 'gray';
  }
}

// TODO: add hover on time line rows with details
// and remove details from the label

export default defineComponent({
  props: {
    event: {
      type: Object as PropType<NormalizedEvent>,
      required: true,
    },
  },
  computed: {
    transaction(): InspectorTransaction {
      return this.event.payload[0];
    },
    grid() {
      let {duration} = this.transaction;

      const totalCells = 5;
      const width = (duration / totalCells + 1);
      const widthPercent = (100 / (totalCells + 1)).toFixed(2);

      const segments = [duration];
      for (let i = 0; i < totalCells; i++) {
        const d = Math.abs(duration -= width)
        segments.push(Math.floor(d))
      }

      return {
        segments: segments.reverse(),
        width,
        widthPercent
      }
    },
    segmentTypes(): { color: string, type: string }[] {
      return [...new Set(this.segments.map(data => data.type))].map(type => ({
        color: `bg-${segmentColor(type)}-600`,
        type
      }))
    },
    segments(): InspectorSegment[] {
      return this.event.payload
          .filter((i: Inspector): boolean => i.model === 'segment' && this.transaction.hash === i.transaction.hash);
    },
    series(): object[] {
      const {duration} = this.transaction

      return this.segments.map(
          (segment: InspectorSegment) => {
            const widthPercent = Math.max((segment.duration * 100 / duration).toFixed(2), 0.5)
            const marginPercent = (segment.start * 100 / duration).toFixed()

            return {
              widthPercent,
              marginPercent,
              segment,
              color: `bg-${segmentColor(segment.type)}-600`
            }
          }
      );
    },
  }
})
</script>

<style lang="scss" scoped>
@import "assets/mixins";

.waterfall {
  @apply py-5;
}

.waterfall-title {
  @include text-muted;
  @apply font-bold uppercase text-sm mb-5;
}

.waterfall-segment-types {
  @apply flex space-x-7 mb-4;
}

.waterfall-segment-type {
  @apply flex items-center;
}

.waterfall-segment-type__color-box {
  @apply w-4 h-4 rounded mr-2;
}

.waterfall-segment-type__label {
  @apply text-xs font-bold;
}

.waterfall-segments {
  @apply overflow-x-scroll border border-gray-50 dark:border-gray-600;
}

.waterfall-segments-cells {
  @apply grid grid-cols-6 divide-x divide-gray-50 dark:divide-gray-600 border-b border-gray-50 dark:border-gray-600 font-bold text-center text-2xs sm:text-xs md:text-sm;
}

.waterfall-segments-cell {
  @apply py-2 pl-3;
}

.waterfall-series {
  background-image: linear-gradient(to right, f3f3f3 1px, transparent 1px)
}

.dark .waterfall-series {
  background-image: linear-gradient(to right, #ffffff1c 1px, transparent 1px)
}

.waterfall-series__segment {
  @apply mt-5 text-right;
}

.waterfall-series__segment-label {
  @apply text-2xs md:text-xs font-bold whitespace-nowrap;
}

.waterfall-series__segment-start {
  @apply flex items-center justify-end;
  background-color: rgba(255, 255, 255, 0.04);
}

.waterfall-series__segment-end {
  @apply flex-1;
  background-color: rgba(255, 255, 255, 0.04);
}

.waterfall-series__segment-start-label {
  @apply text-2xs font-bold text-gray-200 mr-3;
}

.waterfall-series__segment-time {
  @apply flex-none;
  min-width: 0px;
}

.waterfall-series__segment-start,
.waterfall-series__segment-time,
.waterfall-series__segment-end {
  @apply h-4 md:h-5 lg:h-6;
}
</style>