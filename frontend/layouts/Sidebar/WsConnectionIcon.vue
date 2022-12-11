<template>
  <div :title="title">
    <svg v-if="connected" xmlns="http://www.w3.org/2000/svg"
         style="margin: auto; background: transparent; display: block; shape-rendering: auto;" width="100%"
         height="100%" viewBox="0 0 100 100" preserveAspectRatio="xMidYMid">
      <circle cx="28" cy="75" r="11" fill="#0055a5">
        <animate attributeName="fill-opacity" repeatCount="indefinite" dur="1.1764705882352942s" values="0;1;1" keyTimes="0;0.2;1" begin="0s"></animate>
      </circle>
      <path d="M28 47A28 28 0 0 1 56 75" fill="none" stroke="#2d78bc" stroke-width="10">
        <animate attributeName="stroke-opacity" repeatCount="indefinite" dur="1.1764705882352942s" values="0;1;1" keyTimes="0;0.2;1" begin="0.11764705882352942s"></animate>
      </path>
      <path d="M28 25A50 50 0 0 1 78 75" fill="none" stroke="#45aee7" stroke-width="10">
        <animate attributeName="stroke-opacity" repeatCount="indefinite" dur="1.1764705882352942s" values="0;1;1" keyTimes="0;0.2;1" begin="0.23529411764705885s"></animate>
      </path>
    </svg>
    <svg v-else xmlns="http://www.w3.org/2000/svg" x="0px" y="0px" viewBox="0 0 1000 1000" xml:space="preserve">
        <g><path fill="#F43F3F" d="M370.1,257.7c-131.9,25-258,88.5-360.1,190.4l98,97.8c76.5-76.4,169.9-125.8,268.2-148.1L370.1,257.7z M175.2,612.9l95.2,95c34-33.9,73.7-58.8,116-74.5l-6-139C305.2,514.5,234.2,554,175.2,612.9z M616.5,255.3l-6,139.6C713.6,415.7,812,466,892,545.8l98-97.8C884.4,342.7,753.3,278.4,616.5,255.3z M432.8,735.9c-37.1,37-37.1,97.1,0,134.1c37.1,37,97.3,37,134.4,0c37.1-37,37.1-97.1,0-134.1C530.1,698.8,469.9,698.8,432.8,735.9z M600.3,628.8c47.3,15.3,91.7,41.7,129.3,79.1l95.2-95C762.3,550.6,686.4,510,606.3,491L600.3,628.8z M534.7,632.5l23.1-530.3H430.7l23.1,530.3H534.7z"/></g>
    </svg>
  </div>
</template>

<script>
export default {
  computed: {
    title() {
      return this.connected ? 'Websocket connected' : 'Websocket disconnected'
    },
    connected() {
      return this.$store.getters['ws/connected']
    }
  }
}
</script>
