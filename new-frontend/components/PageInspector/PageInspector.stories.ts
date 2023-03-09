import {Meta, Story} from "@storybook/vue3";
import {normalizeInspectorEvent} from "~/utils/normalize-event";
import inspectorEventMock from '~/mocks/inspector.json'
import PageInspector from './PageInspector.vue';

export default {
  title: "Pages/Inspector",
  component: PageInspector
} as Meta<typeof PageInspector>;

const Template: Story = (args) => ({
  components: {PageInspector},
  setup() {
    return {
      args,
    };
  },
  template: `<PageInspector v-bind="args"/>`,
});

export const Default = Template.bind({});

Default.args = {
  event: normalizeInspectorEvent(inspectorEventMock),
};
