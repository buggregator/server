import { Meta, Story } from "@storybook/vue3";
import { EVENT_TYPES } from "~/config/constants";
import PreviewCard from "~/components/PreviewCard/PreviewCard.vue";

export default {
  title: "Preview/PreviewCard",
  component: PreviewCard
}as Meta<typeof PreviewCard>;

const Template: Story = (args) => ({
  components: { PreviewCard },
  setup() {
    return {
      args,
    };
  },
  template: '<PreviewCard v-bind="args">Hello world!</PreviewCard>',
});

export const Default = Template.bind({});
Default.args = {
  event: {
    id: 'da076402-6f98-4ada-bae2-d77d405cf427',
    type: EVENT_TYPES.MONOLOG,
    serverName: "My server",
    origin: {
      one: 1,
      two: 2,
    },
    date: new Date(1673266869 * 1000),
    labels: ['Monolog', '200' ]
  },
};
