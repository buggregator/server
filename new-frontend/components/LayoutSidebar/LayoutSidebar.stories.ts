import { Meta, Story } from "@storybook/vue3";
import LayoutSidebar from "~/components/LayoutSidebar/LayoutSidebar.vue";

export default {
  title: "Layouts/LayoutSidebar",
  component: LayoutSidebar
}as Meta<typeof LayoutSidebar>;

const Template: Story = (args) => ({
  components: { LayoutSidebar },
  setup() {
    return {
      args,
    };
  },
  template: '<div style="width: 100px;"><layout-sidebar v-bind="args" /></div>',
});

export const Default = Template.bind({});
Default.args = {
  isConnected: true
};
