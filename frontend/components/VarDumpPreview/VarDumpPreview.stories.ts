import { Meta, Story } from "@storybook/vue3";
import { normalizeVarDumpEvent } from "~/utils/normalize-event";

import varDumpObjectEventMock from '~/mocks/var-dump-object.json'
import varDumpNumberEventMock from '~/mocks/var-dump-number.json'
import varDumpStringEventMock from '~/mocks/var-dump-string.json'
import varDumpArrayEventMock from '~/mocks/var-dump-array.json'
import VarDumpPreview from '~/components/VarDumpPreview/VarDumpPreview.vue';

export default {
  title: "VarDump/Components/Preview",
  component: VarDumpPreview
} as Meta<typeof VarDumpPreview>;

const Template: Story = (args) => ({
  components: { VarDumpPreview },
  setup() {
    return {
      args,
    };
  },
  template: `<VarDumpPreview v-bind="args" />`,
});

export const Object = Template.bind({});

Object.args = {
  event: normalizeVarDumpEvent(varDumpObjectEventMock),
};

export const Number = Template.bind({});

Number.args = {
  event: normalizeVarDumpEvent(varDumpNumberEventMock),
};

export const String = Template.bind({});

String.args = {
  event: normalizeVarDumpEvent(varDumpStringEventMock),
};

export const Array = Template.bind({});

Array.args = {
  event: normalizeVarDumpEvent(varDumpArrayEventMock),
};
