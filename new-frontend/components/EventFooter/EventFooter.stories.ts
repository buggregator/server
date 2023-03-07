import { Meta, Story } from "@storybook/vue3";
import EventFooter from "~/components/EventFooter/EventFooter.vue";

export default {
  title: "Event/EventFooter",
  component: EventFooter,
}as Meta<typeof EventFooter>;

const Template: Story = (args: typeof Object) => ({
  components: { EventFooter },
  setup() {
    return {
      args,
    };
  },
  template: '<event-footer v-bind="args" />',
});

export const Default = Template.bind({});
Default.args = {
  serverName: "My server",
  originConfig: {
    one: 1,
    two: 2,
  },
};
