<template>
  <div class="email-attachments__list">
    <div v-for="(attachment, i) in attachments" class="email-attachments__list-item">
      <span class="email-attachments__item-index">{{ i + 1 }}.</span>
      <span class="email-attachments__item-name">{{ attachment.name }}</span>
    </div>
  </div>
</template>

<script lang="ts">
import {defineComponent, PropType} from "vue";

export default defineComponent({
  props: {
    attachments: {
      type: Array,
      required: true
    }
  },
})
</script>

<style lang="scss" scoped>
.email-attachments__list {
  @apply flex flex-col divide-y text-sm;
}

.email-attachments__list-item {
  @apply py-2;
}

.email-attachments__item-index {

}

.email-attachments__item-name {

}
</style>
