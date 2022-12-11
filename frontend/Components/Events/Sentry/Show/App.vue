<template>
    <section class="py-5 px-4 md:px-6 lg:px-8 border-b" v-if="event.contexts.app">
        <h3 class="text-muted font-bold uppercase text-sm mb-5">app</h3>

        <Table>
            <TableRow v-if="event.contexts.app.app_build" :title="'App Build'">
                {{ event.contexts.app.app_build }}
            </TableRow>

            <TableRow v-if="event.contexts.app.app_identifier" :title="'Build ID'">
                {{ event.contexts.app.app_identifier }}
            </TableRow>

            <TableRow v-if="event.contexts.app.app_name" :title="'Build Name'">
                {{ event.contexts.app.app_name }}
            </TableRow>

            <TableRow v-if="event.contexts.app.app_start_time" :title="'Start Time'">
                {{ format_date(event.contexts.app.app_start_time) }}
            </TableRow>

            <TableRow v-if="event.contexts.app.app_version" :title="'Version'">
                {{ event.contexts.app.app_version }}
            </TableRow>

            <TableRow v-if="event.contexts.app.permissions" :title="'Permissions'">
                <CodeSnippet class="mt-3" language="json" :code="event.contexts.app.permissions"/>
            </TableRow>
        </Table>
    </section>
</template>

<script>
import Table from "@/Components/UI/Table"
import TableRow from "@/Components/UI/TableRow"
import CodeSnippet from "@/Components/UI/CodeSnippet"
export default {
    components: {
        Table, TableRow, CodeSnippet
    },
    props: {
        event: Object
    },
    methods: {
        format_date(value){
            if (value) {
                return new Date(parseInt(value)).toString()
            }
        },
    }
}
</script>
