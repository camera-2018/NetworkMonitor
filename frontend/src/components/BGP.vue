<!-- eslint-disable @typescript-eslint/no-explicit-any -->
<script lang="ts" setup>
import { use } from 'echarts/core'
import { CanvasRenderer } from 'echarts/renderers'
import { GraphChart } from 'echarts/charts'
import { TooltipComponent, TitleComponent } from 'echarts/components'
import { ECElementEvent, ElementEvent } from 'echarts'
import { ref, computed, watch } from 'vue'
import { Netmask } from 'netmask'
import { getBetweenness, getBGP, getCloseness } from '../api/bgp'
import { prettierNet } from '../utils/colornet'
import { ASData } from '../api/meta'
import BGPUptime from './uptime/BGPUptime.vue'
import { useDark } from '@vueuse/core'
import { mergeObjects } from '../utils/obj'
import { useGraph, useGraphEvent } from '@/state/graph'
import { useRoute, onBeforeRouteLeave } from 'vue-router'
import { dispatchEchartAction } from '@/state/graph'
import { useASMeta } from '@/state/meta'
import { fontColor } from '@/state/font'
import { setUpdatedTime } from '@/state/updated_time'

const isDark = useDark()

const ASMeta = useASMeta()

interface Edge {
  source: string
  target: string
  value: number
  lineStyle?: any
  symbol?: string[]
  emphasis?: any
}

interface Node {
  name: string
  value: string
  meta?: any
  peer_num: number
  betweenness?: number
  closeness?: number
  symbolSize?: number
  symbol?: string
  network: string[]
  itemStyle?: object
}

interface Params<T> {
  dataType: string
  data: T
}

use([CanvasRenderer, GraphChart, TooltipComponent, TitleComponent])
const { option, selectList, loading: graphLoading } = useGraph()

const route = useRoute()
const name = computed<string>(() => route.params.name as string)

import { updatedData } from '@/state/event'
watch(updatedData, (data) => {
  if (data?.type === 'bgp' && data.key === name.value) loadData(name.value)
})

graphLoading.value = true

option.title = {
  text: 'DN11 & Vidar Network',
  textStyle: {
    color: fontColor,
  },
  subtext: computed(
    () =>
      (nodes.value &&
        edges.value &&
        `Nodes: ${nodes.value.reduce(
          (p, c) => p + (c.peer_num === 0 ? 0 : 1),
          0,
        )} Peers: ${edges.value.length}`) ||
      '',
  ),
}
option.tooltip = {
  trigger: 'item',
  triggerOn: 'mousemove',
  backgroundColor: computed(() => (isDark.value ? '#333' : 'white')),
  textStyle: {
    color: computed(() => (isDark.value ? 'white' : 'black')),
  },
  confine: true,
  enterable: true,
  formatter: (params: Params<any>) => {
    if (params.dataType === 'edge') {
      params = params as Params<Edge>
      return `${params.data.source} ↔ ${params.data.target}`
    }

    // dataType === node
    params = params as Params<Node>
    let output = `ASN: ${params.data.name}`

    if (params.data.meta) {
      const metadata: ASData['metadata'][''] = params.data.meta
      if (metadata.display) {
        output += `<br/>Name: ${metadata.display}`
      }
      if (metadata?.monitor?.appendix) {
        const {
          monitor: { appendix },
        } = metadata
        for (let key in appendix) {
          const value = appendix[key] as string | string[]
          if (typeof value === 'string') {
            output += `<br/>${key}: ${value}`
          } else if (Array.isArray(value)) {
            output += `<br/>${key}:`
            for (let i in value) {
              output += `<br/> - ${value[i]}`
            }
          }
        }
      }
    }
    if (params.data.betweenness !== undefined) {
      output += `<br/>Betweenness: ${params.data.betweenness.toFixed(3)}`
    }
    if (params.data.closeness !== undefined) {
      output += `<br/>Closeness: ${params.data.closeness.toFixed(3)}`
    }
    output += `<br/>Network:<br/>`
    if (ASMeta.value) {
      output += prettierNet(
        params.data.network,
        params.data.name,
        ASMeta.value.announcements,
      )
    } else {
      output += params.data.network.join('<br/>')
      output += `<br/>`
    }
    output += `Peer Count: <div class="peer_count"> ${params.data.peer_num} </div>`
    return output
  },
  position: function () {
    return [20, 50]
  },
}

option.series = [
  {
    type: 'graph',
    symbolSize: 50,
    layout: 'force',
    lineStyle: {
      color: 'source',
      opacity: 0.4,
      width: 0.5,
      curveness: 0.1,
    },
    force: {
      repulsion: 500,
      gravity: 0.3,
      friction: 1,
      edgeLength: [10, 140],
      layoutAnimation: false,
    },
    roam: true,
    label: {
      show: true,
      position: 'right',
      color: 'inherit',
      fontWeight: 1000,
      fontFamily: 'Microsoft YaHei',
      formatter: (params: any) => {
        if (params.data.meta?.display) {
          return params.data.meta.display
        }
        return params.data.name
      },
    },
    labelLayout: {
      hideOverlap: true,
    },
    edgeLabel: {
      show: false,
    },
    draggable: true,
    data: [],
    links: [],
    emphasis: {
      focus: 'adjacency',
      lineStyle: {
        width: 10,
      },
    },
  },
]

const bgpData = ref<Awaited<ReturnType<typeof getBGP>>>()
const betweenness = ref<Awaited<ReturnType<typeof getBetweenness>>>()
const closeness = ref<Awaited<ReturnType<typeof getCloseness>>>()
const nodes = computed(() =>
  bgpData.value?.as
    .reduce((nodes, cur) => {
      nodes.push({
        name: cur.asn.toString(),
        value: cur.asn.toString(),
        peer_num: 0,
        betweenness: betweenness.value?.[cur.asn.toString()] || 0,
        closeness: closeness.value?.[cur.asn.toString()] || 0,
        network: cur.network
          .sort((a, b) => parseInt(a.split('/')[1]) - parseInt(b.split('/')[1]))
          .reduce(
            (network, cur) =>
              network.findIndex((net) => {
                let nmask = new Netmask(net)
                return nmask.contains(cur) || nmask.toString() === cur
              }) === -1
                ? [...network, cur]
                : network,
            [] as string[],
          )
          .sort((a, b) => {
            let an = a.split(/[./]/).map((x) => parseInt(x))
            let bn = b.split(/[./]/).map((x) => parseInt(x))
            for (let i = 0; i < an.length; i++) {
              if (an[i] > bn[i]) {
                return 1
              } else if (an[i] < bn[i]) {
                return -1
              }
            }
            return -1
          }),
      })
      return nodes
    }, [] as Node[])
    .map((node) => {
      node.peer_num =
        bgpData.value?.link.filter((lk) => {
          return (
            lk.src === parseInt(node.name) || lk.dst === parseInt(node.name)
          )
        }).length || 0
      node.value = '' + node.peer_num
      node.symbolSize = Math.pow(node.peer_num + 3, 1 / 2) * 7

      const nodeMeta = ASMeta.value?.metadata?.[node.name]
      if (nodeMeta?.monitor?.customNode) {
        mergeObjects(node, nodeMeta.monitor?.customNode)
      }
      if (nodeMeta) {
        node.meta = nodeMeta
      }

      if (node.peer_num === 0) {
        node.symbol =
          'path://M255.633,0C145.341,0.198,55.994,89.667,55.994,200.006v278.66c0,14.849,17.953,22.285,28.453,11.786l38.216-39.328 l54.883,55.994c6.51,6.509,17.063,6.509,23.572,0L256,451.124l54.883,55.994c6.509,6.509,17.062,6.509,23.571,0l54.884-55.994 l38.216,39.327c10.499,10.499,28.453,3.063,28.453-11.786V201.719C456.006,91.512,365.84-0.197,255.633,0z M172.664,266.674 c-27.572,0-50.001-22.429-50.001-50.001s22.43-50.001,50.001-50.001s50.001,22.43,50.001,50.001S200.236,266.674,172.664,266.674z M339.336,266.674c-27.572,0-50.001-22.429-50.001-50.001s22.43-50.001,50.001-50.001s50.001,22.43,50.001,50.001 S366.908,266.674,339.336,266.674z'
      }
      return node
    }),
)

const edges = computed(() =>
  bgpData.value?.link.reduce((edges, cur) => {
    const src = nodes.value?.find((node) => node.name === cur.src.toString())
    const dst = nodes.value?.find((node) => node.name === cur.dst.toString())
    if (src == null || dst == null) {
      return edges
    }
    edges.push({
      source: cur.src.toString(),
      target: cur.dst.toString(),
      value: 100 / Math.min(src.peer_num, dst.peer_num) + 10,
    })
    return edges
  }, [] as Edge[]),
)

// update graph
watch([nodes, edges], async () => {
  if (!nodes.value || !edges.value) {
    return
  }
  const setLoadingOnce = (() => {
    let once = false
    return () => {
      if (once) return
      once = true
      option.series[0].force.friction = 1
      return
    }
  })()

  // remove not existed edges
  for (let i = 0; i < option.series[0].links.length; i++) {
    if (
      edges.value?.findIndex(
        (edge) =>
          edge.source === option.series[0].links[i].source &&
          edge.target === option.series[0].links[i].target,
      ) === -1
    ) {
      setLoadingOnce()
      option.series[0].links.splice(i, 1)
      i--
    }
  }

  // refresh nodes
  for (let i = 0; i < option.series[0].data.length; i++) {
    const idx = nodes.value?.findIndex(
      (node) => node.name === option.series[0].data[i].name,
    )
    if (idx === -1) {
      setLoadingOnce()
      option.series[0].data.splice(i, 1)
      i--
      continue
    }

    if (option.series[0].data[i].peer_num !== nodes.value[idx].peer_num)
      option.series[0].data[i].peer_num = nodes.value[idx].peer_num

    if (
      option.series[0].data[i].network.join('|') !==
      nodes.value[idx].network.join('|')
    )
      option.series[0].data[i].network = nodes.value[idx].network

    if (
      JSON.stringify(option.series[0].data[i].meta) !==
      JSON.stringify(nodes.value[idx].meta)
    ) {
      option.series[0].data[i] = nodes.value[idx]
    }
  }

  // add new nodes
  for (let i = 0; i < nodes.value.length; i++) {
    if (
      option.series[0].data.findIndex(
        (node: Node) => node.name === nodes.value?.[i].name,
      ) === -1
    ) {
      setLoadingOnce()
      option.series[0].data.push(nodes.value[i])
    }
  }
  // add new edges
  for (let i = 0; i < edges.value.length; i++) {
    if (
      option.series[0].links.findIndex(
        (edge: Edge) =>
          edge.source === edges.value?.[i].source &&
          edge.target === edges.value?.[i].target,
      ) === -1
    ) {
      setLoadingOnce()
      option.series[0].links.push(edges.value[i])
    }
  }

  option.series[0].force.edgeLength[1] = nodes.value.length * 3.5
  option.series[0].force.friction = 0.15
  if (bgpData.value?.updated_at) setUpdatedTime(bgpData.value?.updated_at)
  graphLoading.value = false
})

// update selectList
watch([nodes, ASMeta], () => {
  if (!nodes.value) return
  selectList.value = nodes.value.map((n) => {
    return {
      label: n.meta?.display || n.name,
      asn: n.name,
      name: n.name,
      display: n.meta?.display || n.name,
      network: [
        ...n.network,
        ...(ASMeta.value?.announcements.assigned
          .filter((a) => a.asn === n.name)
          .map((a) => a.prefix) || []),
      ],
      value: n.name,
      onselected: () => {
        dispatchEchartAction({
          type: 'highlight',
          seriesIndex: 0,
          name: n.name,
        })
        dispatchEchartAction({
          type: 'showTip',
          seriesIndex: 0,
          name: n.name,
        })
      },
    }
  })
})

async function loadData(name: string) {
  closeness.value = await getCloseness(name)
  betweenness.value = await getBetweenness(name)
  bgpData.value = await getBGP(name)
}
watch(name, loadData, {
  immediate: true,
})

const interval = setInterval(() => {
  loadData(name.value)
}, 60 * 1000)

let timer: ReturnType<typeof setTimeout>

const { handleClick, handleMouseDown, handleMouseUp, handleZrClick } =
  useGraphEvent()
handleMouseDown.value = () => {
  if (timer) {
    clearTimeout(timer)
  }
  option.series[0].force.friction = 0.15
  option.series[0].force.layoutAnimation = true
}

handleMouseUp.value = () => {
  timer = setTimeout(() => {
    option.series[0].force.layoutAnimation = false
  }, 6000)
}

const uptime_asn = ref(0)

handleClick.value = (e: ECElementEvent) => {
  if (e.dataType === 'node') {
    const data = e.data as Node
    uptime_asn.value = parseInt(data.name)
  }
}

handleZrClick.value = (e: ElementEvent) => {
  if (e.target === undefined) {
    uptime_asn.value = 0
  }
}

onBeforeRouteLeave(() => {
  if (timer) {
    clearTimeout(timer)
  }
  if (interval) {
    clearInterval(interval)
  }
  uptime_asn.value = 0
})
</script>

<template>
  <Transition name="fade" appear>
    <BGPUptime
      class="uptime"
      v-if="uptime_asn !== 0"
      :asn="uptime_asn"
      :grName="name"
    />
  </Transition>
</template>

<style scoped>
.uptime {
  position: absolute;
  top: 0;
  left: 0;
  right: 0;
  bottom: 0;
  width: 80vw;
  height: 80vh;
  margin: auto;
}

.fade-enter-active {
  transition: all 0.2s ease-in;
}
.fade-leave-active {
  transition: all 0.2s ease-out;
}

.fade-enter-from {
  opacity: 0;
}

.fade-enter,
.fade-leave-to {
  opacity: 0;
}
</style>
