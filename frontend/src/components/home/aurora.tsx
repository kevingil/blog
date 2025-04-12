'use client'

import { motion } from "framer-motion";
import React from "react";
import { AuroraBackground } from "@/components/ui/aurora-background";
import { useState } from "react";
import { useEffect } from "react";
import { usePathname } from "next/navigation";


export default function Aurora() {

   const [dimmed, setDimmed] = useState<boolean>(true);
   let pathname = usePathname();
   
   useEffect(() => {
    if (pathname === '/') {
      setDimmed(false);
    } else {
      setDimmed(true);
    }
   }, [pathname]);


    return (
    <div className={`fixed top-0 left-0 w-full h-full body-background`}>
    <AuroraBackground className={` ${dimmed ? 'opacity-20' : 'opacity-100'} animate-opacity`}>
      <motion.div
        initial={{ opacity: 0.0, y: 40 }}
        whileInView={{ opacity: 1, y: 0 }}
        transition={{
          delay: 0.3,
          duration: 4,
          ease: "easeInOut",
        }}
        className={`relative flex flex-col gap-4 items-center justify-center px-4`}
      >
      </motion.div>
    </AuroraBackground>

    </div>
  );
}
