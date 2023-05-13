import { Meta, Story } from "@storybook/vue3";

import ValueDump from '~/components/ValueDump/ValueDump.vue';
import varDumpObjectEventMock from '~/mocks/var-dump-object.json'

export default {
  title: "Components/ValueDump",
  component: ValueDump
} as Meta<typeof ValueDump>;

const Template: Story = (args) => ({
  components: { ValueDump },
  setup() {
    return {
      args,
    };
  },
  template: `<ValueDump v-bind="args" />`,
});

export const String = Template.bind({});

String.args = {
  value: `&lt;?xml&nbsp;version="1.0"?&gt;<br>&lt;one&gt;<br>&nbsp;&nbsp;&lt;two&gt;<br>&nbsp;&nbsp;&nbsp;&nbsp;&lt;three&gt;3&lt;/three&gt;<br>&nbsp;&nbsp;&lt;/two&gt;<br>&lt;/one&gt;`,
  type: 'string',
};

export const Boolean = Template.bind({});

Boolean.args = {
  value: true,
};

export const SfDump = Template.bind({});

SfDump.args = {
  value: varDumpObjectEventMock.payload.payload.value,
};
