import { Meta, Story } from "@storybook/vue3";
import FileView from "~/components/FileView/FileView.vue";

export default {
  title: "Components/FileView",
  component: FileView
} as Meta<typeof FileView>;

const Template: Story = (args) => ({
  components: { FileView },
  setup() {
    return {
      args,
    };
  },
  template: `<FileView :file="args.file">
  This is a row 1
  </FileView>`,
});

export const FileDefault = Template.bind({});
FileDefault.args = {
  file: {
    file_name: "/root/repos/buggreagtor/examples/app/Modules/Ray/RayCommon.php",
    line_number: 96,
    class: "App\\Http\\Controllers\\CallAction",
    method: "rayTrace",
    vendor_frame: false,
  }
};
