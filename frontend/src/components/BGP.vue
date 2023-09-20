<script setup lang="ts">
import { ref, onMounted } from 'vue';
import ForceGraph3D from '3d-force-graph';
import { GraphData, LinkObject, NodeObject } from 'three-forcegraph';
import { reactive } from "vue";
import { Netmask } from "netmask";
import { UnrealBloomPass } from 'three/examples/jsm/postprocessing/UnrealBloomPass.js';
import { Object3D } from 'three';
import SpriteText from "three-spritetext";
const graph = ref<HTMLElement | null>(null);  // 创建一个 ref
const nodeSprites = new Map<string, SpriteText>();
import { getASMetaData, getBGP } from "../api/graph";
const infoBox = ref<HTMLDivElement | null>(null);

const hoveredNode = ref<Node | null>(null);
type Edge = LinkObject & {
  source: string;
  target: string;
  value: number
  lineStyle?: any;
  symbol?: string[];
}

type Node = NodeObject & {
  name: string;
  value: string;
  meta?: any;
  peer_num: number;
  symbolSize?: number;
  network: string[];
  itemStyle?: any;
}

function mergeObjects(obj1: any, obj2: any): any {
  for (const key in obj2) {
    if (
      obj2.hasOwnProperty(key)
      && (obj1.hasOwnProperty(key) || !(key in obj1))
    ) {
      if (typeof obj2[key] === 'object' && obj2[key] !== null && typeof obj1[key] === 'object' && obj1[key] !== null) {
        mergeObjects(obj1[key], obj2[key]);
      } else {
        obj1[key] = obj2[key];
      }
    }
  }
}


// 在组件装载完成后执行初始化
onMounted(async () => {
  let gData = reactive<GraphData>({
    nodes: [] as NodeObject[],
    links: [] as LinkObject[],
  });
  await getBGP().then((resp) => {
    if (!resp.data.as) {
      alert("no data")
      return
    }

    const nodes = resp.data.as.reduce((nodes, cur) => {
      nodes.push({
        name: cur.asn.toString(),
        id: cur.asn.toString(),
        value: cur.asn.toString(),
        peer_num: 0,
        network: cur.network.sort((a, b) =>
          parseInt(a.split("/")[1]) - parseInt(b.split("/")[1])
        ).reduce((network, cur) =>
          network.findIndex((net) => {
            let nmask = new Netmask(net);
            return nmask.contains(cur) || nmask.toString() === cur;
          }) === -1 ?
            [...network, cur] : network
          , [] as string[]
        ).sort((a, b) => {
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
        })
      })
      return nodes;
    }, [] as Node[]);

    nodes.forEach(async (node) => {
      node = reactive(node);
      node.peer_num = resp.data.link.filter((lk) => {
        return lk.src === parseInt(node.name) || lk.dst === parseInt(node.name);
      }).length;
      node.value = '' + node.peer_num;
      node.symbolSize = Math.pow(node.peer_num, 1 / 2) * 7;
      node.itemStyle = {
        shadowBlur: Math.pow(node.peer_num, 1 / 2) * 2,
      }
      getASMetaData(parseInt(node.name)).catch((e) => {
        if (e.response.status !== 404) {
          console.log(e)
        }
      }).then((resp) => {
        if (resp === undefined) {
          return
        }
        if (resp.customNode) {
          mergeObjects(node, resp.customNode)
        }
        node.meta = resp;

        const sprite = nodeSprites.get(node.name);
        if (sprite) {
          sprite.text = node.meta?.display || node.name;
        }
      });
    });

    const edges = resp.data.link.reduce((edges, cur) => {
      const src = nodes.find((node) => node.name === cur.src.toString());
      const dst = nodes.find((node) => node.name === cur.dst.toString());
      if (src == null || dst == null) {
        return edges;
      }
      edges.push({
        source: cur.src.toString(),
        target: cur.dst.toString(),
        value: 1 / Math.pow(Math.min(src.peer_num, dst.peer_num), 1 / 2) * 100,
      });
      return edges;
    }, [] as Edge[]);

    gData = {
      nodes: nodes,
      links: edges
    };
  });
  console.log(gData)

  if (graph.value) {
    const myGraph = ForceGraph3D({ antialias: true, alpha: true });
    myGraph(graph.value)
      .nodeOpacity(0.7)
      .linkResolution(32)
      .linkOpacity(0.3)
      .enableNavigationControls(true)
      .nodeLabel("")
      .nodeColor(() => {
        return "#138385"
      })
      .nodeResolution(32)
      .nodeRelSize(2)
      .backgroundColor('#000003')
      .cooldownTicks(900)
      .nodeThreeObject((node: Node) => {
        const sprite = new SpriteText(node.name);
        nodeSprites.set(node.name, sprite);
        const group = new Object3D();
        sprite.material.depthWrite = false;
        sprite.color = "#ff6f00";
        sprite.textHeight = 1.3;
        sprite.strokeWidth = "1";
        sprite.strokeColor = "#000000";
        sprite.renderOrder = 999;
        sprite.material.depthTest = false;
        group.add(sprite);
        return group;
      })
      .nodeThreeObjectExtend(true)
      // 显示一个跟随鼠标的小窗口 里面是节点的信息

      .onNodeHover(function (node: NodeObject | null) {
        hoveredNode.value = node ? node as Node : null;
      })

      .onNodeClick((node: Node) => {
          // Aim at node from outside it
          const distance = 40;
          const distRatio = 1 + distance/Math.hypot(node.x, node.y, node.z);

          const newPos = node.x || node.y || node.z
            ? { x: node.x * distRatio, y: node.y * distRatio, z: node.z * distRatio }
            : { x: 0, y: 0, z: distance }; // special case if node is in (0,0,0)

          myGraph.cameraPosition(
            newPos, // new position
            node, // lookAt ({ x, y, z })
            3000  // ms transition duration
          );
        })

      .onNodeDrag(function (node: NodeObject | null) {
        myGraph.nodeColor(() => {
          return "#138385"
        })
        if (node) {
          myGraph.nodeColor((tnode: NodeObject) => {
            if (node.id === tnode.id) {
              return "#ffbbff"
            }
            return "#138385"
          })
          myGraph.linkColor((link: LinkObject) => {
            if (link.source.id === node.id || link.target.id === node.id) {
              return "#ffbbff"
            }
            return "#ffffff"
          })
        }
      })
      .graphData(gData)
    const bloomPass = new UnrealBloomPass();
    bloomPass.strength = 2.3;
    bloomPass.radius = 1;
    bloomPass.threshold = 0;
    myGraph.postProcessingComposer().addPass(bloomPass);
  }

}
);
</script>

<template>
  <div ref="graph"></div>
  <Transition>
  <div ref="infoBox" class="info-box" v-if="hoveredNode">
    <p>Name: {{ hoveredNode.name }}</p>
    <p>Peer Num: {{ hoveredNode.peer_num }}</p>
    <p>Network: {{ hoveredNode.network.join(", ") }}</p>
    <p>Meta: {{ hoveredNode.meta.display }}</p>
  </div>
</Transition>
</template>


<style scoped>

.info-box {
  position: absolute;
  top: 1rem;
  left: 1rem;
  width: 200px;
  background-color: #cccccc;
  border-radius: 5px;
  padding: 1rem;
  box-shadow: 0 0 10px rgba(0, 0, 0, 0.2);
  pointer-events: none;
  z-index: 999;
  font-size: 1rem;
  color: #000;
  font-family: sans-serif;
  line-height: 1.5;
  text-align: left;
  opacity: 0.9;
  overflow: auto;
  word-break: break-all;
  word-wrap: break-word;
  white-space: pre-wrap;
  -webkit-user-select: none;
  -moz-user-select: none;
  -ms-user-select: none;
  user-select: none;
}

.v-enter-active,
.v-leave-active {
  transition: opacity 0.5s ease;
}

.v-enter-from,
.v-leave-to {
  opacity: 0;
}

</style>
