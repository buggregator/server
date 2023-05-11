import { Meta, Story } from "@storybook/vue3";
import { normalizeHttpDumpEvent } from "~/utils/normalize-event";
import httpDumpEventMock from '~/mocks/http-dump.json'
import HttpDumpPage from "~/components/HttpDumpPage/HttpDumpPage.vue";

export default {
  title: "HttpDump/Page/HttpDumpPage",
  component: HttpDumpPage
} as Meta<typeof HttpDumpPage>;

const Template: Story = (args) => ({
  components: { HttpDumpPage },
  setup() {
    return {
      args,
    };
  },
  template: `<HttpDumpPage v-bind="args" />`,
});

export const Default = Template.bind({});

const normalizeEvent = normalizeHttpDumpEvent(httpDumpEventMock)

Default.args = {
  event: normalizeEvent,
};
