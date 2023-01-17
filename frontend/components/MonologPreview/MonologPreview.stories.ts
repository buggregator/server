import { Meta, Story } from "@storybook/vue3";
import { normalizeMonologEvent } from "~/utils/normalize-event";
import monologEventMock from '~/mocks/monolog.json'
import MonologPreview from '~/components/MonologPreview/MonologPreview.vue';

export default {
  title: "Monolog/Components/Preview",
  component: MonologPreview
} as Meta<typeof MonologPreview>;

const Template: Story = (args) => ({
  components: { MonologPreview },
  setup() {
    return {
      args,
    };
  },
  template: `<MonologPreview v-bind="args" />`,
});

export const Event = Template.bind({});

Event.args = {
  event: normalizeMonologEvent(monologEventMock),
};
