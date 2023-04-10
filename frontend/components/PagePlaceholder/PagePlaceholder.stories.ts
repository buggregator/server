import { Meta, Story } from "@storybook/vue3";
import PagePlaceholder from "~/components/PagePlaceholder/PagePlaceholder.vue";

export default {
  title: "Components/PagePlaceholder",
  component: PagePlaceholder
}as Meta<typeof PagePlaceholder>;

const Template: Story = (args) => ({
  components: { PagePlaceholder },
  setup() {
    return {
      args,
    };
  },
  template: '<PagePlaceholder v-bind="args" />',
});

export const Default = Template.bind({});
Default.args = {
};
