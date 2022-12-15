<template>
  <canvas class="flamechart"></canvas>
</template>

<script>
import FlameChart from 'flame-chart-js'

export default {
  props: {
    event: Object,
  },
  mounted() {
    const canvas = this.$el
    canvas.width = '100%';
    canvas.height = 400;

    const flameChart = new FlameChart({
      canvas, // mandatory
      data: [
        {
          name: 'foo',
          start: 300,
          duration: 200,
          type: 'task',
          children: [
            {
              name: 'foo',
              start: 310,
              duration: 50,
              type: 'sub-task',
              color: '#AA0000'
            }
          ]
        }
      ],
      marks: [
        {
          shortName: 'DCL',
          fullName: 'DOMContentLoaded',
          timestamp: 500
        }
      ],
      waterfall: { /* ... */ },
      colors: {
        'task': '#FFFFFF',
        'sub-task': '#000000'
      },
      settings: {
        options: {
          timeUnits: 'ms'
        },
      }
    });
  },
  destroyed() {

  }
}
</script>
