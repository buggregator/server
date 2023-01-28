import { EVENT_TYPES } from "~/config/constants";
import { Meta, Story } from "@storybook/vue3";
import timeFormat from "~/utils/timeFormat";
import CodeSnippet from "./CodeSnippet.vue";

export default {
  title: "Components/CodeSnippet",
  component: CodeSnippet,
} as Meta<typeof CodeSnippet>;

const Template: Story = (args) => ({
  components: { CodeSnippet },
  setup() {
    return {
      args,
    };
  },
  template: `<code-snippet v-bind="args" />`,
});

export const Default = Template.bind({});
Default.args = {
  code: {
    id: 'da076402-6f98-4ada-bae2-d77d405cf427',
    type: EVENT_TYPES.MONOLOG,
    serverName: "My server",
    origin: {
      one: 1,
      two: 2,
    },
    date: timeFormat(new Date(1673266869 * 1000)),
    labels: ['Monolog', '200' ]
  },
  language: 'javascript'
};
